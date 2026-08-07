package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type memoryImageGenerationJobRepository struct {
	mu    sync.Mutex
	jobs  map[string]ImageGenerationJob
	order []string
}

func newMemoryImageGenerationJobRepository() *memoryImageGenerationJobRepository {
	return &memoryImageGenerationJobRepository{jobs: make(map[string]ImageGenerationJob)}
}

func (r *memoryImageGenerationJobRepository) Create(_ context.Context, job *ImageGenerationJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = cloneImageGenerationJob(*job)
	r.order = append(r.order, job.ID)
	return nil
}

func (r *memoryImageGenerationJobRepository) CountActiveForUser(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, job := range r.jobs {
		if job.UserID == userID && (job.Status == ImageGenerationJobStatusPending || job.Status == ImageGenerationJobStatusRunning) {
			count++
		}
	}
	return count, nil
}

func (r *memoryImageGenerationJobRepository) ListForUser(_ context.Context, userID int64, limit int) ([]ImageGenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]ImageGenerationJob, 0)
	for index := len(r.order) - 1; index >= 0 && len(items) < limit; index-- {
		job := r.jobs[r.order[index]]
		if job.UserID == userID {
			items = append(items, cloneImageGenerationJob(job))
		}
	}
	return items, nil
}

func (r *memoryImageGenerationJobRepository) GetForUser(_ context.Context, userID int64, jobID string) (*ImageGenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok || job.UserID != userID {
		return nil, os.ErrNotExist
	}
	cloned := cloneImageGenerationJob(job)
	return &cloned, nil
}

func (r *memoryImageGenerationJobRepository) ClaimNext(_ context.Context, _ time.Duration) (*ImageGenerationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range r.order {
		job := r.jobs[id]
		if job.Status != ImageGenerationJobStatusPending {
			continue
		}
		now := time.Now()
		job.Status = ImageGenerationJobStatusRunning
		job.StartedAt = &now
		job.UpdatedAt = now
		job.AttemptCount++
		r.jobs[id] = job
		cloned := cloneImageGenerationJob(job)
		return &cloned, nil
	}
	return nil, nil
}

func (r *memoryImageGenerationJobRepository) MarkSucceeded(_ context.Context, jobID string, results []ImageGenerationJobResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok {
		return errors.New("job not found")
	}
	now := time.Now()
	job.Status = ImageGenerationJobStatusSucceeded
	job.Results = append([]ImageGenerationJobResult(nil), results...)
	job.FinishedAt = &now
	job.UpdatedAt = now
	r.jobs[jobID] = job
	return nil
}

func (r *memoryImageGenerationJobRepository) Requeue(_ context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok {
		return errors.New("job not found")
	}
	job.Status = ImageGenerationJobStatusPending
	job.StartedAt = nil
	job.UpdatedAt = time.Now()
	r.jobs[jobID] = job
	return nil
}

func (r *memoryImageGenerationJobRepository) MarkFailed(_ context.Context, jobID string, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok {
		return errors.New("job not found")
	}
	now := time.Now()
	job.Status = ImageGenerationJobStatusFailed
	job.ErrorMessage = message
	job.FinishedAt = &now
	job.UpdatedAt = now
	r.jobs[jobID] = job
	return nil
}

func cloneImageGenerationJob(job ImageGenerationJob) ImageGenerationJob {
	job.RequestBody = append(json.RawMessage(nil), job.RequestBody...)
	job.Results = append([]ImageGenerationJobResult(nil), job.Results...)
	return job
}

func TestImageGenerationJobServiceCompletesAndStoresImage(t *testing.T) {
	repo := newMemoryImageGenerationJobRepository()
	cfg := &config.Config{ImageGenerationJobs: config.ImageGenerationJobsConfig{
		WorkerCount:         1,
		MaxQueuedPerUser:    2,
		PollIntervalSeconds: 1,
		TaskTimeoutSeconds:  5,
		StoragePath:         t.TempDir(),
		MaxImageBytes:       1 << 20,
	}}
	svc := NewImageGenerationJobService(repo, cfg)
	png := []byte("\x89PNG\r\n\x1a\nimage-data")
	svc.SetExecutor(func(_ context.Context, userID int64, requestBody []byte) ([]byte, error) {
		require.Equal(t, int64(42), userID)
		require.JSONEq(t, `{"model":"gpt-image-2"}`, string(requestBody))
		return json.Marshal(map[string]any{"data": []map[string]any{{
			"b64_json":      base64.StdEncoding.EncodeToString(png),
			"output_format": "png",
		}}})
	})
	require.NoError(t, svc.Start())
	t.Cleanup(svc.Stop)

	job, err := svc.Submit(context.Background(), CreateImageGenerationJobInput{
		UserID:      42,
		Model:       "gpt-image-2",
		Prompt:      "draw",
		Size:        "1024x1024",
		Quality:     "low",
		ImageCount:  1,
		RequestBody: json.RawMessage(`{"model":"gpt-image-2"}`),
	})
	require.NoError(t, err)
	require.Equal(t, ImageGenerationJobStatusPending, job.Status)

	require.Eventually(t, func() bool {
		current, getErr := svc.GetForUser(context.Background(), 42, job.ID)
		return getErr == nil && current.Status == ImageGenerationJobStatusSucceeded
	}, 3*time.Second, 20*time.Millisecond)

	completed, err := svc.GetForUser(context.Background(), 42, job.ID)
	require.NoError(t, err)
	require.Len(t, completed.Results, 1)
	require.Equal(t, "image/png", completed.Results[0].MimeType)
	path, mimeType, err := svc.ImageFile(context.Background(), 42, job.ID, 0)
	require.NoError(t, err)
	require.Equal(t, "image/png", mimeType)
	stored, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, png, stored)
}

func TestImageGenerationJobServiceEnforcesUserQueueLimit(t *testing.T) {
	repo := newMemoryImageGenerationJobRepository()
	cfg := &config.Config{ImageGenerationJobs: config.ImageGenerationJobsConfig{MaxQueuedPerUser: 1}}
	svc := NewImageGenerationJobService(repo, cfg)
	input := CreateImageGenerationJobInput{
		UserID: 7, Model: "gpt-image-2", Prompt: "draw", Size: "auto", Quality: "auto", ImageCount: 1,
		RequestBody: json.RawMessage(`{"model":"gpt-image-2"}`),
	}
	_, err := svc.Submit(context.Background(), input)
	require.NoError(t, err)
	_, err = svc.Submit(context.Background(), input)
	require.ErrorIs(t, err, ErrTooManyImageGenerationJobs)
}

func TestImageGenerationJobServiceRequeuesRunningJobOnShutdown(t *testing.T) {
	repo := newMemoryImageGenerationJobRepository()
	cfg := &config.Config{ImageGenerationJobs: config.ImageGenerationJobsConfig{
		WorkerCount: 1, PollIntervalSeconds: 1, TaskTimeoutSeconds: 30, StoragePath: t.TempDir(),
	}}
	svc := NewImageGenerationJobService(repo, cfg)
	svc.SetExecutor(func(ctx context.Context, _ int64, _ []byte) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	require.NoError(t, svc.Start())
	job, err := svc.Submit(context.Background(), CreateImageGenerationJobInput{
		UserID: 9, Model: "gpt-image-2", Prompt: "draw", Size: "auto", Quality: "auto", ImageCount: 1,
		RequestBody: json.RawMessage(`{"model":"gpt-image-2"}`),
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		current, getErr := svc.GetForUser(context.Background(), 9, job.ID)
		return getErr == nil && current.Status == ImageGenerationJobStatusRunning
	}, 2*time.Second, 20*time.Millisecond)

	svc.Stop()
	current, err := svc.GetForUser(context.Background(), 9, job.ID)
	require.NoError(t, err)
	require.Equal(t, ImageGenerationJobStatusPending, current.Status)
}
