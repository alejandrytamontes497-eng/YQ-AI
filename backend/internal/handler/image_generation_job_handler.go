package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const imageGenerationJobRequestMaxBytes = 1 << 20

type ImageGenerationJobHandler struct {
	jobs                *service.ImageGenerationJobService
	gateway             *GatewayHandler
	openAIGateway       *OpenAIGatewayHandler
	subscriptionService *service.SubscriptionService
}

type imageGenerationJobRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
	N       int    `json:"n"`
}

type imageGenerationJobResultResponse struct {
	Index         int    `json:"index"`
	URL           string `json:"url"`
	MimeType      string `json:"mime_type"`
	ByteSize      int64  `json:"byte_size,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type imageGenerationJobResponse struct {
	ID           string                             `json:"id"`
	Status       string                             `json:"status"`
	Model        string                             `json:"model"`
	Prompt       string                             `json:"prompt"`
	Size         string                             `json:"size"`
	Quality      string                             `json:"quality"`
	ImageCount   int                                `json:"image_count"`
	Results      []imageGenerationJobResultResponse `json:"results"`
	ErrorMessage string                             `json:"error_message,omitempty"`
	AttemptCount int                                `json:"attempt_count"`
	StartedAt    *time.Time                         `json:"started_at,omitempty"`
	FinishedAt   *time.Time                         `json:"finished_at,omitempty"`
	CreatedAt    time.Time                          `json:"created_at"`
	UpdatedAt    time.Time                          `json:"updated_at"`
}

func NewImageGenerationJobHandler(
	jobs *service.ImageGenerationJobService,
	gateway *GatewayHandler,
	openAIGateway *OpenAIGatewayHandler,
	subscriptionService *service.SubscriptionService,
) (*ImageGenerationJobHandler, error) {
	h := &ImageGenerationJobHandler{
		jobs:                jobs,
		gateway:             gateway,
		openAIGateway:       openAIGateway,
		subscriptionService: subscriptionService,
	}
	if jobs == nil || gateway == nil || openAIGateway == nil {
		return nil, errors.New("image generation job handler dependencies are not ready")
	}
	jobs.SetExecutor(h.execute)
	if err := jobs.Start(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *ImageGenerationJobHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, imageGenerationJobRequestMaxBytes))
	if err != nil {
		response.BadRequest(c, "Invalid image generation request")
		return
	}
	var request imageGenerationJobRequest
	var payload map[string]any
	if len(body) == 0 || json.Unmarshal(body, &request) != nil || json.Unmarshal(body, &payload) != nil {
		response.BadRequest(c, "Invalid image generation request")
		return
	}
	request.Model = strings.TrimSpace(request.Model)
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.Size = strings.TrimSpace(request.Size)
	request.Quality = strings.TrimSpace(request.Quality)
	if !service.IsOpenAIImageGenerationModel(request.Model) {
		response.BadRequest(c, "Images endpoint requires an image model")
		return
	}
	if request.Prompt == "" {
		response.BadRequest(c, "Image prompt is required")
		return
	}
	if len(request.Prompt) > 32000 {
		response.BadRequest(c, "Image prompt is too long")
		return
	}
	if request.N == 0 {
		request.N = 1
	}
	if request.N < 1 || request.N > 4 {
		response.BadRequest(c, "Image count must be between 1 and 4")
		return
	}
	if request.Size == "" {
		request.Size = "auto"
	}
	if request.Quality == "" {
		request.Quality = "auto"
	}
	payload["model"] = request.Model
	payload["prompt"] = request.Prompt
	payload["size"] = request.Size
	payload["quality"] = request.Quality
	payload["n"] = request.N
	payload["response_format"] = "b64_json"
	delete(payload, "stream")
	canonicalBody, err := json.Marshal(payload)
	if err != nil {
		response.BadRequest(c, "Invalid image generation request")
		return
	}

	job, err := h.jobs.Submit(c.Request.Context(), service.CreateImageGenerationJobInput{
		UserID:      subject.UserID,
		Model:       request.Model,
		Prompt:      request.Prompt,
		Size:        request.Size,
		Quality:     request.Quality,
		ImageCount:  request.N,
		RequestBody: canonicalBody,
	})
	if errors.Is(err, service.ErrTooManyImageGenerationJobs) {
		response.Error(c, http.StatusTooManyRequests, err.Error())
		return
	}
	if err != nil {
		response.InternalError(c, "Failed to create image generation job")
		return
	}
	response.Accepted(c, h.jobResponse(*job))
}

func (h *ImageGenerationJobHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	jobs, err := h.jobs.ListForUser(c.Request.Context(), subject.UserID, limit)
	if err != nil {
		response.InternalError(c, "Failed to load image generation jobs")
		return
	}
	items := make([]imageGenerationJobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, h.jobResponse(job))
	}
	response.Success(c, items)
}

func (h *ImageGenerationJobHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	job, err := h.jobs.GetForUser(c.Request.Context(), subject.UserID, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, "Image generation job not found")
		return
	}
	if err != nil {
		response.InternalError(c, "Failed to load image generation job")
		return
	}
	response.Success(c, h.jobResponse(*job))
}

func (h *ImageGenerationJobHandler) Events(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "Streaming is unavailable")
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	writeEvent := func(name string, data any) bool {
		payload, err := json.Marshal(data)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", name, payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	lastSnapshotVersion := ""
	writeSnapshot := func(force bool) bool {
		jobs, err := h.jobs.ListForUser(c.Request.Context(), subject.UserID, 24)
		if err != nil {
			return true
		}
		var version strings.Builder
		for _, job := range jobs {
			fmt.Fprintf(&version, "%s:%s:%d;", job.ID, job.Status, job.UpdatedAt.UnixNano())
		}
		if !force && version.String() == lastSnapshotVersion {
			return true
		}
		lastSnapshotVersion = version.String()
		items := make([]imageGenerationJobResponse, 0, len(jobs))
		for _, job := range jobs {
			items = append(items, h.jobResponse(job))
		}
		return writeEvent("snapshot", items)
	}
	if !writeSnapshot(true) {
		return
	}

	events, unsubscribe := h.jobs.Subscribe(subject.UserID)
	defer unsubscribe()
	reconcileTicker := time.NewTicker(5 * time.Second)
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer reconcileTicker.Stop()
	defer heartbeatTicker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, open := <-events:
			if !open || !writeEvent("job", h.jobResponse(event.Job)) {
				return
			}
		case <-reconcileTicker.C:
			if !writeSnapshot(false) {
				return
			}
		case <-heartbeatTicker.C:
			if _, err := io.WriteString(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *ImageGenerationJobHandler) Image(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		response.NotFound(c, "Generated image not found")
		return
	}
	path, mimeType, err := h.jobs.ImageFile(c.Request.Context(), subject.UserID, c.Param("id"), index)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, os.ErrNotExist) {
		response.NotFound(c, "Generated image not found")
		return
	}
	if err != nil {
		response.InternalError(c, "Failed to load generated image")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		response.NotFound(c, "Generated image not found")
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		response.InternalError(c, "Failed to load generated image")
		return
	}
	c.Header("Content-Type", mimeType)
	c.Header("Cache-Control", "private, max-age=86400")
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}

func (h *ImageGenerationJobHandler) execute(ctx context.Context, userID int64, requestBody []byte) ([]byte, error) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", bytes.NewReader(requestBody)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})

	if !h.gateway.BindUserImageGenerationContext(c, h.subscriptionService) {
		return nil, imageGenerationExecutionError(recorder)
	}
	h.openAIGateway.Images(c)
	if recorder.Code < 200 || recorder.Code >= 300 {
		return nil, imageGenerationExecutionError(recorder)
	}
	return append([]byte(nil), recorder.Body.Bytes()...), nil
}

func imageGenerationExecutionError(recorder *httptest.ResponseRecorder) error {
	message := strings.TrimSpace(recorder.Body.String())
	var payload struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(recorder.Body.Bytes(), &payload) == nil {
		if strings.TrimSpace(payload.Error.Message) != "" {
			message = payload.Error.Message
		} else if strings.TrimSpace(payload.Message) != "" {
			message = payload.Message
		}
	}
	if len(message) > 1000 {
		message = message[:1000]
	}
	return &service.ImageGenerationJobExecutionError{StatusCode: recorder.Code, Message: message}
}

func (h *ImageGenerationJobHandler) jobResponse(job service.ImageGenerationJob) imageGenerationJobResponse {
	results := make([]imageGenerationJobResultResponse, 0, len(job.Results))
	for _, result := range job.Results {
		url := result.UpstreamURL
		if result.FileName != "" {
			url = fmt.Sprintf("/user/images/jobs/%s/images/%d", job.ID, result.Index)
		}
		results = append(results, imageGenerationJobResultResponse{
			Index:         result.Index,
			URL:           url,
			MimeType:      result.MimeType,
			ByteSize:      result.ByteSize,
			RevisedPrompt: result.RevisedPrompt,
		})
	}
	return imageGenerationJobResponse{
		ID:           job.ID,
		Status:       job.Status,
		Model:        job.Model,
		Prompt:       job.Prompt,
		Size:         job.Size,
		Quality:      job.Quality,
		ImageCount:   job.ImageCount,
		Results:      results,
		ErrorMessage: job.ErrorMessage,
		AttemptCount: job.AttemptCount,
		StartedAt:    job.StartedAt,
		FinishedAt:   job.FinishedAt,
		CreatedAt:    job.CreatedAt,
		UpdatedAt:    job.UpdatedAt,
	}
}
