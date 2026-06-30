package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const callAuditCaptureBytes = 4096

type callAuditResponseWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *callAuditResponseWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *callAuditResponseWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *callAuditResponseWriter) capture(data []byte) {
	if len(data) == 0 || w.buf.Len() >= callAuditCaptureBytes {
		return
	}
	remaining := callAuditCaptureBytes - w.buf.Len()
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.buf.Write(data)
}

func CallAuditMiddleware(auditService *service.CallAuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auditService == nil {
			c.Next()
			return
		}

		requestPrefix := captureRequestPrefix(c)
		wrapped := &callAuditResponseWriter{ResponseWriter: c.Writer}
		c.Writer = wrapped

		c.Next()

		apiKey, ok := middleware.GetAPIKeyFromContext(c)
		if !ok || apiKey == nil {
			return
		}
		subject, ok := middleware.GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			return
		}

		var accountID *int64
		if raw, exists := c.Get(opsAccountIDKey); exists {
			switch v := raw.(type) {
			case int64:
				if v > 0 {
					accountID = &v
				}
			case int:
				if v > 0 {
					v64 := int64(v)
					accountID = &v64
				}
			}
		}

		status := c.Writer.Status()
		if status <= 0 {
			status = http.StatusOK
		}

		groupID := apiKey.GroupID
		path := c.FullPath()
		if path == "" && c.Request != nil {
			path = c.Request.URL.Path
		}
		method := ""
		if c.Request != nil {
			method = c.Request.Method
		}
		model := strings.TrimSpace(gjson.GetBytes(requestPrefix, "model").String())
		requestID := c.GetHeader("x-request-id")

		input := service.CreateCallAuditLogInput{
			UserID:          subject.UserID,
			APIKeyID:        apiKey.ID,
			AccountID:       accountID,
			GroupID:         groupID,
			RequestID:       requestID,
			Method:          method,
			Path:            path,
			Model:           model,
			StatusCode:      status,
			RequestExcerpt:  service.CallAuditExcerpt(requestPrefix),
			ResponseExcerpt: service.CallAuditExcerpt(wrapped.buf.Bytes()),
			CreatedAt:       time.Now(),
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = auditService.Create(ctx, input)
		}()
	}
}

func captureRequestPrefix(c *gin.Context) []byte {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return nil
	}
	limited := io.LimitReader(c.Request.Body, callAuditCaptureBytes)
	prefix, _ := io.ReadAll(limited)
	c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(prefix), c.Request.Body))
	return prefix
}
