package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseImageGenerationJobCreateRequestMultipart(t *testing.T) {
	body, multipartContentType := imageGenerationJobTestMultipart(t)
	for _, contentType := range []string{multipartContentType, "application/json"} {
		t.Run(contentType, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/jobs", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", contentType)

			request, payload, reference, err := parseImageGenerationJobCreateRequest(c)
			require.NoError(t, err)
			require.Equal(t, "gpt-image-2", request.Model)
			require.Equal(t, "follow this composition", request.Prompt)
			require.Equal(t, 1, request.N)
			require.Equal(t, "gpt-image-2", payload["model"])
			require.NotNil(t, reference)
			require.Equal(t, "reference.png", reference.OriginalName)
			require.Equal(t, "image/png", reference.MimeType)

			model, err := userImageGenerationModel(contentType, body)
			require.NoError(t, err)
			require.Equal(t, "gpt-image-2", model)
		})
	}
}

func TestParseImageGenerationJobCreateRequestRejectsJSONEncodedImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/jobs", strings.NewReader(
		`{"model":"gpt-image-2","prompt":"draw","image":{}}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	_, _, _, err := parseImageGenerationJobCreateRequest(c)
	var requestErr *imageGenerationJobRequestError
	require.ErrorAs(t, err, &requestErr)
	require.Equal(t, http.StatusUnsupportedMediaType, requestErr.statusCode)
	require.Contains(t, requestErr.message, "multipart/form-data")
}

func TestBuildImageEditMultipart(t *testing.T) {
	imageData := []byte("\x89PNG\r\n\x1a\nreference-data")
	path := t.TempDir() + "/reference.png"
	require.NoError(t, os.WriteFile(path, imageData, 0o600))
	job := &service.ImageGenerationJob{
		Model: "gpt-image-2", Prompt: "follow this composition", Size: "1024x1024",
		Quality: "low", ImageCount: 1, ReferenceImageOriginalName: "reference.png",
		ReferenceImageMimeType: "image/png",
	}

	body, contentType, err := buildImageEditMultipart(job, path)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", contentType)
	parsed, err := (&service.OpenAIGatewayService{}).ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.True(t, parsed.IsEdits())
	require.Equal(t, "gpt-image-2", parsed.Model)
	require.Equal(t, "follow this composition", parsed.Prompt)
	require.Equal(t, "url", parsed.ResponseFormat)
	require.Len(t, parsed.Uploads, 1)
	require.Equal(t, "image/png", parsed.Uploads[0].ContentType)
	require.Equal(t, imageData, parsed.Uploads[0].Data)

	model, err := userImageGenerationModel(contentType, body)
	require.NoError(t, err)
	require.Equal(t, "gpt-image-2", model)
}

func imageGenerationJobTestMultipart(t *testing.T) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "follow this composition"))
	require.NoError(t, writer.WriteField("size", "1024x1024"))
	require.NoError(t, writer.WriteField("quality", "low"))
	require.NoError(t, writer.WriteField("n", "1"))
	part, err := writer.CreateFormFile("image", "reference.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("\x89PNG\r\n\x1a\nreference-data"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body.Bytes(), writer.FormDataContentType()
}
