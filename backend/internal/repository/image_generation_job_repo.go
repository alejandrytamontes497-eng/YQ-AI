package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type imageGenerationJobRepository struct {
	db *sql.DB
}

func NewImageGenerationJobRepository(db *sql.DB) service.ImageGenerationJobRepository {
	return &imageGenerationJobRepository{db: db}
}

func (r *imageGenerationJobRepository) Create(ctx context.Context, job *service.ImageGenerationJob) error {
	results, err := json.Marshal(job.Results)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO image_generation_jobs (
			id, user_id, status, model, prompt, size, quality, image_count,
			request_body, reference_image_file_name, reference_image_original_name,
			reference_image_mime_type, results, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
	`, job.ID, job.UserID, job.Status, job.Model, job.Prompt, job.Size, job.Quality,
		job.ImageCount, []byte(job.RequestBody), job.ReferenceImageFileName,
		job.ReferenceImageOriginalName, job.ReferenceImageMimeType, results, job.CreatedAt)
	return err
}

func (r *imageGenerationJobRepository) CountActiveForUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM image_generation_jobs
		WHERE user_id = $1 AND status IN ($2, $3)
	`, userID, service.ImageGenerationJobStatusPending, service.ImageGenerationJobStatusRunning).Scan(&count)
	return count, err
}

func (r *imageGenerationJobRepository) ListForUser(ctx context.Context, userID int64, limit int) ([]service.ImageGenerationJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, status, model, prompt, size, quality, image_count,
			request_body, reference_image_file_name, reference_image_original_name,
			reference_image_mime_type, results, error_message, attempt_count,
			started_at, finished_at, created_at, updated_at
		FROM image_generation_jobs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	jobs := make([]service.ImageGenerationJob, 0)
	for rows.Next() {
		job, err := scanImageGenerationJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

func (r *imageGenerationJobRepository) GetForUser(ctx context.Context, userID int64, jobID string) (*service.ImageGenerationJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, status, model, prompt, size, quality, image_count,
			request_body, reference_image_file_name, reference_image_original_name,
			reference_image_mime_type, results, error_message, attempt_count,
			started_at, finished_at, created_at, updated_at
		FROM image_generation_jobs
		WHERE id = $1 AND user_id = $2
	`, jobID, userID)
	return scanImageGenerationJob(row)
}

func (r *imageGenerationJobRepository) ClaimNext(ctx context.Context, staleAfter time.Duration) (*service.ImageGenerationJob, error) {
	staleSeconds := int64(staleAfter.Seconds())
	if staleSeconds <= 0 {
		staleSeconds = 360
	}
	row := r.db.QueryRowContext(ctx, `
		WITH next AS (
			SELECT id
			FROM image_generation_jobs
			WHERE status = $1
				OR (status = $2 AND updated_at < NOW() - ($3 * interval '1 second'))
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE image_generation_jobs AS jobs
		SET status = $2,
			started_at = NOW(),
			finished_at = NULL,
			error_message = NULL,
			attempt_count = attempt_count + 1,
			updated_at = NOW()
		FROM next
		WHERE jobs.id = next.id
		RETURNING jobs.id, jobs.user_id, jobs.status, jobs.model, jobs.prompt,
			jobs.size, jobs.quality, jobs.image_count, jobs.request_body,
			jobs.reference_image_file_name, jobs.reference_image_original_name,
			jobs.reference_image_mime_type, jobs.results,
			jobs.error_message, jobs.attempt_count, jobs.started_at, jobs.finished_at,
			jobs.created_at, jobs.updated_at
	`, service.ImageGenerationJobStatusPending, service.ImageGenerationJobStatusRunning, staleSeconds)
	job, err := scanImageGenerationJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *imageGenerationJobRepository) MarkSucceeded(ctx context.Context, jobID string, results []service.ImageGenerationJobResult) error {
	payload, err := json.Marshal(results)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE image_generation_jobs
		SET status = $1, results = $2, error_message = NULL,
			finished_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`, service.ImageGenerationJobStatusSucceeded, payload, jobID)
	return err
}

func (r *imageGenerationJobRepository) Requeue(ctx context.Context, jobID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE image_generation_jobs
		SET status = $1, started_at = NULL, finished_at = NULL,
			error_message = NULL, updated_at = NOW()
		WHERE id = $2 AND status = $3
	`, service.ImageGenerationJobStatusPending, jobID, service.ImageGenerationJobStatusRunning)
	return err
}

func (r *imageGenerationJobRepository) MarkFailed(ctx context.Context, jobID string, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE image_generation_jobs
		SET status = $1, error_message = $2,
			finished_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`, service.ImageGenerationJobStatusFailed, message, jobID)
	return err
}

type imageGenerationJobScanner interface {
	Scan(dest ...any) error
}

func scanImageGenerationJob(scanner imageGenerationJobScanner) (*service.ImageGenerationJob, error) {
	var job service.ImageGenerationJob
	var requestBody []byte
	var resultsJSON []byte
	var errorMessage sql.NullString
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := scanner.Scan(
		&job.ID, &job.UserID, &job.Status, &job.Model, &job.Prompt,
		&job.Size, &job.Quality, &job.ImageCount, &requestBody,
		&job.ReferenceImageFileName, &job.ReferenceImageOriginalName,
		&job.ReferenceImageMimeType, &resultsJSON,
		&errorMessage, &job.AttemptCount, &startedAt, &finishedAt,
		&job.CreatedAt, &job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	job.RequestBody = append(json.RawMessage(nil), requestBody...)
	if len(resultsJSON) > 0 {
		if err := json.Unmarshal(resultsJSON, &job.Results); err != nil {
			return nil, fmt.Errorf("parse image job results: %w", err)
		}
	}
	if job.Results == nil {
		job.Results = []service.ImageGenerationJobResult{}
	}
	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		value := startedAt.Time
		job.StartedAt = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		job.FinishedAt = &value
	}
	return &job, nil
}
