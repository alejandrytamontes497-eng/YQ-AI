package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	imageGenerationJobJSONMaxBytes   = 1 << 20
	imageGenerationReferenceMaxBytes = 20 << 20
	imageGenerationMultipartMaxBytes = imageGenerationReferenceMaxBytes + imageGenerationJobJSONMaxBytes
)

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

type imageGenerationJobRequestError struct {
	statusCode int
	message    string
}

func (e *imageGenerationJobRequestError) Error() string { return e.message }

type imageGenerationJobResultResponse struct {
	Index         int    `json:"index"`
	URL           string `json:"url"`
	MimeType      string `json:"mime_type"`
	ByteSize      int64  `json:"byte_size,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type imageGenerationJobResponse struct {
	ID                     string                             `json:"id"`
	Status                 string                             `json:"status"`
	Model                  string                             `json:"model"`
	Prompt                 string                             `json:"prompt"`
	Size                   string                             `json:"size"`
	Quality                string                             `json:"quality"`
	ImageCount             int                                `json:"image_count"`
	ReferenceImageName     string                             `json:"reference_image_name,omitempty"`
	ReferenceImageMimeType string                             `json:"reference_image_mime_type,omitempty"`
	Results                []imageGenerationJobResultResponse `json:"results"`
	ErrorMessage           string                             `json:"error_message,omitempty"`
	AttemptCount           int                                `json:"attempt_count"`
	StartedAt              *time.Time                         `json:"started_at,omitempty"`
	FinishedAt             *time.Time                         `json:"finished_at,omitempty"`
	CreatedAt              time.Time                          `json:"created_at"`
	UpdatedAt              time.Time                          `json:"updated_at"`
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

func parseImageGenerationJobCreateRequest(c *gin.Context) (
	imageGenerationJobRequest,
	map[string]any,
	*service.ImageGenerationReferenceImage,
	error,
) {
	var request imageGenerationJobRequest
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, imageGenerationMultipartMaxBytes))
	if err != nil {
		return request, nil, nil, &imageGenerationJobRequestError{http.StatusRequestEntityTooLarge, "Image generation request is too large"}
	}
	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	boundary, isMultipart := imageGenerationMultipartBoundary(contentType, body)
	if !isMultipart {
		if len(body) > imageGenerationJobJSONMaxBytes {
			return request, nil, nil, &imageGenerationJobRequestError{http.StatusRequestEntityTooLarge, "Image generation request is too large"}
		}
		var payload map[string]any
		if len(body) == 0 || json.Unmarshal(body, &request) != nil || json.Unmarshal(body, &payload) != nil {
			return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Invalid image generation request"}
		}
		return request, payload, nil, nil
	}

	values := make(map[string]string)
	var referenceImage *service.ImageGenerationReferenceImage
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Invalid multipart image request"}
		}
		name := strings.TrimSpace(part.FormName())
		if part.FileName() != "" {
			if name != "image" || referenceImage != nil {
				_ = part.Close()
				return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Upload exactly one reference image"}
			}
			data, readErr := io.ReadAll(io.LimitReader(part, imageGenerationReferenceMaxBytes+1))
			_ = part.Close()
			if readErr != nil || len(data) == 0 {
				return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Reference image is empty"}
			}
			if len(data) > imageGenerationReferenceMaxBytes {
				return request, nil, nil, &imageGenerationJobRequestError{http.StatusRequestEntityTooLarge, "Reference image must be 20 MB or smaller"}
			}
			detected := strings.ToLower(http.DetectContentType(data))
			if detected != "image/png" && detected != "image/jpeg" && detected != "image/webp" {
				return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Reference image must be PNG, JPEG, or WebP"}
			}
			referenceImage = &service.ImageGenerationReferenceImage{
				OriginalName: part.FileName(),
				MimeType:     detected,
				Data:         data,
			}
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(part, imageGenerationJobJSONMaxBytes))
		_ = part.Close()
		if readErr != nil {
			return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Invalid multipart image request"}
		}
		values[name] = string(data)
	}
	if referenceImage == nil {
		return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Reference image is required for multipart requests"}
	}
	request.Model = values["model"]
	request.Prompt = values["prompt"]
	request.Size = values["size"]
	request.Quality = values["quality"]
	if rawN := strings.TrimSpace(values["n"]); rawN != "" {
		request.N, err = strconv.Atoi(rawN)
		if err != nil {
			return request, nil, nil, &imageGenerationJobRequestError{http.StatusBadRequest, "Image count must be a number"}
		}
	}
	payload := map[string]any{
		"model": request.Model, "prompt": request.Prompt, "size": request.Size,
		"quality": request.Quality, "n": request.N,
	}
	return request, payload, referenceImage, nil
}

func imageGenerationMultipartBoundary(contentType string, body []byte) (string, bool) {
	mediaType, params, _ := mime.ParseMediaType(strings.TrimSpace(contentType))
	if strings.EqualFold(mediaType, "multipart/form-data") {
		if boundary := strings.TrimSpace(params["boundary"]); boundary != "" {
			return boundary, true
		}
	}
	if !bytes.HasPrefix(body, []byte("--")) {
		return "", false
	}
	lineEnd := bytes.Index(body, []byte("\r\n"))
	if lineEnd <= 2 || lineEnd > 202 {
		return "", false
	}
	boundary := strings.TrimSpace(string(body[2:lineEnd]))
	if boundary == "" || strings.ContainsAny(boundary, "\r\n") {
		return "", false
	}
	return boundary, true
}

func (h *ImageGenerationJobHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	request, payload, referenceImage, err := parseImageGenerationJobCreateRequest(c)
	if err != nil {
		var requestErr *imageGenerationJobRequestError
		if errors.As(err, &requestErr) {
			response.Error(c, requestErr.statusCode, requestErr.message)
			return
		}
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
		UserID:         subject.UserID,
		Model:          request.Model,
		Prompt:         request.Prompt,
		Size:           request.Size,
		Quality:        request.Quality,
		ImageCount:     request.N,
		RequestBody:    canonicalBody,
		ReferenceImage: referenceImage,
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

func (h *ImageGenerationJobHandler) execute(ctx context.Context, job *service.ImageGenerationJob) ([]byte, error) {
	if job == nil {
		return nil, errors.New("image generation job is missing")
	}
	requestBody := []byte(job.RequestBody)
	requestPath := "/api/v1/user/images/generations"
	contentType := "application/json"
	if job.ReferenceImageFileName != "" {
		path, err := h.jobs.ReferenceImageFile(job)
		if err != nil {
			return nil, fmt.Errorf("load reference image: %w", err)
		}
		requestBody, contentType, err = buildImageEditMultipart(job, path)
		if err != nil {
			return nil, err
		}
		requestPath = "/api/v1/user/images/edits"
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, requestPath, bytes.NewReader(requestBody)).WithContext(ctx)
	request.Header.Set("Content-Type", contentType)
	c.Request = request
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: job.UserID})

	if !h.gateway.BindUserImageGenerationContext(c, h.subscriptionService) {
		return nil, imageGenerationExecutionError(recorder)
	}
	h.openAIGateway.Images(c)
	if recorder.Code < 200 || recorder.Code >= 300 {
		return nil, imageGenerationExecutionError(recorder)
	}
	return append([]byte(nil), recorder.Body.Bytes()...), nil
}

func buildImageEditMultipart(job *service.ImageGenerationJob, imagePath string) ([]byte, string, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, "", fmt.Errorf("read reference image: %w", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"model":           job.Model,
		"prompt":          job.Prompt,
		"size":            job.Size,
		"quality":         job.Quality,
		"n":               strconv.Itoa(job.ImageCount),
		"response_format": "b64_json",
	}
	for _, name := range []string{"model", "prompt", "size", "quality", "n", "response_format"} {
		if err := writer.WriteField(name, fields[name]); err != nil {
			return nil, "", fmt.Errorf("write image edit field: %w", err)
		}
	}
	fileName := job.ReferenceImageOriginalName
	if strings.TrimSpace(fileName) == "" {
		fileName = "reference.png"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name": "image", "filename": fileName,
	}))
	header.Set("Content-Type", job.ReferenceImageMimeType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, "", fmt.Errorf("create reference image part: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, "", fmt.Errorf("write reference image part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize image edit request: %w", err)
	}
	return body.Bytes(), writer.FormDataContentType(), nil
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
		ID:                     job.ID,
		Status:                 job.Status,
		Model:                  job.Model,
		Prompt:                 job.Prompt,
		Size:                   job.Size,
		Quality:                job.Quality,
		ImageCount:             job.ImageCount,
		ReferenceImageName:     job.ReferenceImageOriginalName,
		ReferenceImageMimeType: job.ReferenceImageMimeType,
		Results:                results,
		ErrorMessage:           job.ErrorMessage,
		AttemptCount:           job.AttemptCount,
		StartedAt:              job.StartedAt,
		FinishedAt:             job.FinishedAt,
		CreatedAt:              job.CreatedAt,
		UpdatedAt:              job.UpdatedAt,
	}
}
