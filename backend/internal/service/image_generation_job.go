package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	ImageGenerationJobStatusPending   = "pending"
	ImageGenerationJobStatusRunning   = "running"
	ImageGenerationJobStatusSucceeded = "succeeded"
	ImageGenerationJobStatusFailed    = "failed"
)

var ErrTooManyImageGenerationJobs = errors.New("too many pending image generation jobs")

type ImageGenerationJobResult struct {
	Index         int    `json:"index"`
	FileName      string `json:"file_name,omitempty"`
	UpstreamURL   string `json:"upstream_url,omitempty"`
	MimeType      string `json:"mime_type"`
	ByteSize      int64  `json:"byte_size,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageGenerationJob struct {
	ID                         string                     `json:"id"`
	UserID                     int64                      `json:"user_id"`
	Status                     string                     `json:"status"`
	Model                      string                     `json:"model"`
	Prompt                     string                     `json:"prompt"`
	Size                       string                     `json:"size"`
	Quality                    string                     `json:"quality"`
	ImageCount                 int                        `json:"image_count"`
	RequestBody                json.RawMessage            `json:"-"`
	ReferenceImageFileName     string                     `json:"-"`
	ReferenceImageOriginalName string                     `json:"reference_image_name,omitempty"`
	ReferenceImageMimeType     string                     `json:"reference_image_mime_type,omitempty"`
	Results                    []ImageGenerationJobResult `json:"results"`
	ErrorMessage               string                     `json:"error_message,omitempty"`
	AttemptCount               int                        `json:"attempt_count"`
	StartedAt                  *time.Time                 `json:"started_at,omitempty"`
	FinishedAt                 *time.Time                 `json:"finished_at,omitempty"`
	CreatedAt                  time.Time                  `json:"created_at"`
	UpdatedAt                  time.Time                  `json:"updated_at"`
}

type CreateImageGenerationJobInput struct {
	UserID         int64
	Model          string
	Prompt         string
	Size           string
	Quality        string
	ImageCount     int
	RequestBody    json.RawMessage
	ReferenceImage *ImageGenerationReferenceImage
}

type ImageGenerationReferenceImage struct {
	OriginalName string
	MimeType     string
	Data         []byte
}

type ImageGenerationJobRepository interface {
	Create(ctx context.Context, job *ImageGenerationJob) error
	CountActiveForUser(ctx context.Context, userID int64) (int, error)
	ListForUser(ctx context.Context, userID int64, limit int) ([]ImageGenerationJob, error)
	GetForUser(ctx context.Context, userID int64, jobID string) (*ImageGenerationJob, error)
	ClaimNext(ctx context.Context, staleAfter time.Duration) (*ImageGenerationJob, error)
	Requeue(ctx context.Context, jobID string) error
	MarkSucceeded(ctx context.Context, jobID string, results []ImageGenerationJobResult) error
	MarkFailed(ctx context.Context, jobID string, message string) error
}

type ImageGenerationJobExecutor func(ctx context.Context, job *ImageGenerationJob) ([]byte, error)

type ImageGenerationJobEvent struct {
	Job ImageGenerationJob `json:"job"`
}

type ImageGenerationJobService struct {
	repo ImageGenerationJobRepository
	cfg  *config.Config

	executorMu sync.RWMutex
	executor   ImageGenerationJobExecutor

	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	stopOnce  sync.Once
	wake      chan struct{}
	wg        sync.WaitGroup

	subscribersMu sync.RWMutex
	subscribers   map[int64]map[chan ImageGenerationJobEvent]struct{}
}

func NewImageGenerationJobService(repo ImageGenerationJobRepository, cfg *config.Config) *ImageGenerationJobService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ImageGenerationJobService{
		repo:        repo,
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
		wake:        make(chan struct{}, 1),
		subscribers: make(map[int64]map[chan ImageGenerationJobEvent]struct{}),
	}
}

func (s *ImageGenerationJobService) SetExecutor(executor ImageGenerationJobExecutor) {
	if s == nil {
		return
	}
	s.executorMu.Lock()
	s.executor = executor
	s.executorMu.Unlock()
}

func (s *ImageGenerationJobService) Start() error {
	if s == nil || s.repo == nil {
		return errors.New("image generation job service is not ready")
	}
	if err := os.MkdirAll(s.storageDir(), 0o750); err != nil {
		return fmt.Errorf("create image job storage: %w", err)
	}

	workerCount := s.workerCount()
	s.startOnce.Do(func() {
		for i := 0; i < workerCount; i++ {
			s.wg.Add(1)
			go s.worker()
		}
	})
	return nil
}

func (s *ImageGenerationJobService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.cancel()
		s.wg.Wait()
	})
}

func (s *ImageGenerationJobService) Submit(ctx context.Context, input CreateImageGenerationJobInput) (*ImageGenerationJob, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("image generation job service is not ready")
	}
	active, err := s.repo.CountActiveForUser(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if active >= s.maxQueuedPerUser() {
		return nil, ErrTooManyImageGenerationJobs
	}

	now := time.Now()
	job := &ImageGenerationJob{
		ID:          newImageGenerationJobID(),
		UserID:      input.UserID,
		Status:      ImageGenerationJobStatusPending,
		Model:       strings.TrimSpace(input.Model),
		Prompt:      strings.TrimSpace(input.Prompt),
		Size:        strings.TrimSpace(input.Size),
		Quality:     strings.TrimSpace(input.Quality),
		ImageCount:  input.ImageCount,
		RequestBody: append(json.RawMessage(nil), input.RequestBody...),
		Results:     []ImageGenerationJobResult{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if input.ReferenceImage != nil {
		if err := s.storeReferenceImage(job, input.ReferenceImage); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Create(ctx, job); err != nil {
		if job.ReferenceImageFileName != "" {
			_ = os.RemoveAll(filepath.Join(s.storageDir(), fmt.Sprintf("%d", job.UserID), job.ID))
		}
		return nil, err
	}
	s.publish(*job)
	s.signalWorker()
	return job, nil
}

func (s *ImageGenerationJobService) ListForUser(ctx context.Context, userID int64, limit int) ([]ImageGenerationJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	return s.repo.ListForUser(ctx, userID, limit)
}

func (s *ImageGenerationJobService) GetForUser(ctx context.Context, userID int64, jobID string) (*ImageGenerationJob, error) {
	return s.repo.GetForUser(ctx, userID, strings.TrimSpace(jobID))
}

func (s *ImageGenerationJobService) Subscribe(userID int64) (<-chan ImageGenerationJobEvent, func()) {
	ch := make(chan ImageGenerationJobEvent, 16)
	s.subscribersMu.Lock()
	if s.subscribers[userID] == nil {
		s.subscribers[userID] = make(map[chan ImageGenerationJobEvent]struct{})
	}
	s.subscribers[userID][ch] = struct{}{}
	s.subscribersMu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.subscribersMu.Lock()
			delete(s.subscribers[userID], ch)
			if len(s.subscribers[userID]) == 0 {
				delete(s.subscribers, userID)
			}
			s.subscribersMu.Unlock()
			close(ch)
		})
	}
}

func (s *ImageGenerationJobService) ImageFile(ctx context.Context, userID int64, jobID string, index int) (string, string, error) {
	job, err := s.GetForUser(ctx, userID, jobID)
	if err != nil {
		return "", "", err
	}
	for _, result := range job.Results {
		if result.Index != index || result.FileName == "" {
			continue
		}
		path := filepath.Join(s.storageDir(), fmt.Sprintf("%d", userID), job.ID, filepath.Base(result.FileName))
		if _, err := os.Stat(path); err != nil {
			return "", "", err
		}
		return path, result.MimeType, nil
	}
	return "", "", os.ErrNotExist
}

func (s *ImageGenerationJobService) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		job, err := s.repo.ClaimNext(s.ctx, s.staleAfter())
		if err == nil && job != nil {
			s.publish(*job)
			s.execute(job)
			continue
		}

		timer := time.NewTimer(s.pollInterval())
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-s.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (s *ImageGenerationJobService) execute(job *ImageGenerationJob) {
	s.executorMu.RLock()
	executor := s.executor
	s.executorMu.RUnlock()
	if executor == nil {
		s.finishFailed(job, "image generation executor is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, s.taskTimeout())
	defer cancel()
	body, err := executor(ctx, job)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) && errors.Is(s.ctx.Err(), context.Canceled) {
			_ = s.repo.Requeue(context.Background(), job.ID)
			return
		}
		s.finishFailed(job, err.Error())
		return
	}
	results, err := s.persistResponse(job, body)
	if err != nil {
		s.finishFailed(job, err.Error())
		return
	}
	if err := s.repo.MarkSucceeded(context.Background(), job.ID, results); err != nil {
		return
	}
	if completed, err := s.repo.GetForUser(context.Background(), job.UserID, job.ID); err == nil && completed != nil {
		s.publish(*completed)
	}
}

func (s *ImageGenerationJobService) ReferenceImageFile(job *ImageGenerationJob) (string, error) {
	if job == nil || strings.TrimSpace(job.ReferenceImageFileName) == "" {
		return "", os.ErrNotExist
	}
	path := filepath.Join(s.storageDir(), fmt.Sprintf("%d", job.UserID), job.ID, filepath.Base(job.ReferenceImageFileName))
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *ImageGenerationJobService) storeReferenceImage(job *ImageGenerationJob, image *ImageGenerationReferenceImage) error {
	if job == nil || image == nil || len(image.Data) == 0 {
		return errors.New("reference image is empty")
	}
	if len(image.Data) > 20<<20 {
		return errors.New("reference image exceeds 20 MB")
	}
	mimeType := strings.ToLower(strings.TrimSpace(image.MimeType))
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(image.Data)))
	if detected != "image/png" && detected != "image/jpeg" && detected != "image/webp" {
		return errors.New("reference image must be PNG, JPEG, or WebP")
	}
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = detected
	}
	extension := "png"
	switch detected {
	case "image/jpeg":
		extension = "jpg"
	case "image/webp":
		extension = "webp"
	}
	dir := filepath.Join(s.storageDir(), fmt.Sprintf("%d", job.UserID), job.ID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create reference image directory: %w", err)
	}
	fileName := "reference." + extension
	if err := writeFileAtomic(filepath.Join(dir, fileName), image.Data); err != nil {
		return fmt.Errorf("store reference image: %w", err)
	}
	job.ReferenceImageFileName = fileName
	job.ReferenceImageOriginalName = sanitizeReferenceImageName(image.OriginalName, fileName)
	job.ReferenceImageMimeType = detected
	return nil
}

func sanitizeReferenceImageName(value string, fallback string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "" || value == "." {
		return fallback
	}
	if len(value) > 255 {
		value = value[:255]
	}
	return value
}

func (s *ImageGenerationJobService) finishFailed(job *ImageGenerationJob, message string) {
	message = strings.TrimSpace(message)
	if len(message) > 1000 {
		message = message[:1000]
	}
	if message == "" {
		message = "image generation failed"
	}
	if err := s.repo.MarkFailed(context.Background(), job.ID, message); err != nil {
		return
	}
	if failed, err := s.repo.GetForUser(context.Background(), job.UserID, job.ID); err == nil && failed != nil {
		s.publish(*failed)
	}
}

func (s *ImageGenerationJobService) persistResponse(job *ImageGenerationJob, body []byte) ([]ImageGenerationJobResult, error) {
	var payload struct {
		Data []struct {
			B64JSON       string `json:"b64_json"`
			URL           string `json:"url"`
			DownloadURL   string `json:"download_url"`
			RevisedPrompt string `json:"revised_prompt"`
			OutputFormat  string `json:"output_format"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse image generation response: %w", err)
	}
	if len(payload.Data) == 0 {
		return nil, errors.New("image generation returned no images")
	}

	dir := filepath.Join(s.storageDir(), fmt.Sprintf("%d", job.UserID), job.ID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create image result directory: %w", err)
	}
	results := make([]ImageGenerationJobResult, 0, len(payload.Data))
	for index, item := range payload.Data {
		result := ImageGenerationJobResult{Index: index, RevisedPrompt: item.RevisedPrompt}
		if strings.TrimSpace(item.B64JSON) != "" {
			data, err := base64.StdEncoding.DecodeString(item.B64JSON)
			if err != nil {
				return nil, fmt.Errorf("decode generated image %d: %w", index, err)
			}
			if int64(len(data)) > s.maxImageBytes() {
				return nil, fmt.Errorf("generated image %d exceeds storage limit", index)
			}
			format, mimeType := generatedImageFormat(data, item.OutputFormat)
			fileName := fmt.Sprintf("%d.%s", index, format)
			if err := writeFileAtomic(filepath.Join(dir, fileName), data); err != nil {
				return nil, fmt.Errorf("store generated image %d: %w", index, err)
			}
			result.FileName = fileName
			result.MimeType = mimeType
			result.ByteSize = int64(len(data))
		} else {
			result.UpstreamURL = strings.TrimSpace(item.DownloadURL)
			if result.UpstreamURL == "" {
				result.UpstreamURL = strings.TrimSpace(item.URL)
			}
			if result.UpstreamURL == "" {
				return nil, fmt.Errorf("generated image %d has no data", index)
			}
			result.MimeType = outputFormatMimeType(item.OutputFormat)
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *ImageGenerationJobService) publish(job ImageGenerationJob) {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()
	for ch := range s.subscribers[job.UserID] {
		select {
		case ch <- ImageGenerationJobEvent{Job: job}:
		default:
		}
	}
}

func (s *ImageGenerationJobService) signalWorker() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *ImageGenerationJobService) workerCount() int {
	if s.cfg != nil && s.cfg.ImageGenerationJobs.WorkerCount > 0 {
		return s.cfg.ImageGenerationJobs.WorkerCount
	}
	return 2
}

func (s *ImageGenerationJobService) maxQueuedPerUser() int {
	if s.cfg != nil && s.cfg.ImageGenerationJobs.MaxQueuedPerUser > 0 {
		return s.cfg.ImageGenerationJobs.MaxQueuedPerUser
	}
	return 10
}

func (s *ImageGenerationJobService) pollInterval() time.Duration {
	if s.cfg != nil && s.cfg.ImageGenerationJobs.PollIntervalSeconds > 0 {
		return time.Duration(s.cfg.ImageGenerationJobs.PollIntervalSeconds) * time.Second
	}
	return time.Second
}

func (s *ImageGenerationJobService) taskTimeout() time.Duration {
	if s.cfg != nil && s.cfg.ImageGenerationJobs.TaskTimeoutSeconds > 0 {
		return time.Duration(s.cfg.ImageGenerationJobs.TaskTimeoutSeconds) * time.Second
	}
	return 5 * time.Minute
}

func (s *ImageGenerationJobService) staleAfter() time.Duration {
	return s.taskTimeout() + time.Minute
}

func (s *ImageGenerationJobService) maxImageBytes() int64 {
	if s.cfg != nil && s.cfg.ImageGenerationJobs.MaxImageBytes > 0 {
		return s.cfg.ImageGenerationJobs.MaxImageBytes
	}
	return 64 << 20
}

func (s *ImageGenerationJobService) storageDir() string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.ImageGenerationJobs.StoragePath) != "" {
		return filepath.Clean(s.cfg.ImageGenerationJobs.StoragePath)
	}
	if info, err := os.Stat("/app/data"); err == nil && info.IsDir() {
		return "/app/data/image-jobs"
	}
	return filepath.Clean("./data/image-jobs")
}

func newImageGenerationJobID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err == nil {
		return "imgjob_" + hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("imgjob_%d", time.Now().UnixNano())
}

func generatedImageFormat(data []byte, fallback string) (string, string) {
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return "png", "image/png"
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "jpg", "image/jpeg"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "webp", "image/webp"
	}
	format := strings.ToLower(strings.TrimSpace(fallback))
	switch format {
	case "jpg", "jpeg":
		return "jpg", "image/jpeg"
	case "webp":
		return "webp", "image/webp"
	default:
		return "png", "image/png"
	}
}

func outputFormatMimeType(format string) string {
	_, mimeType := generatedImageFormat(nil, format)
	return mimeType
}

func writeFileAtomic(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".image-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o640); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

type ImageGenerationJobExecutionError struct {
	StatusCode int
	Message    string
}

func (e *ImageGenerationJobExecutionError) Error() string {
	if e == nil || strings.TrimSpace(e.Message) == "" {
		return http.StatusText(http.StatusBadGateway)
	}
	return e.Message
}
