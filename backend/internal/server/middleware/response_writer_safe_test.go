package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type panicStatusWriter struct {
	gin.ResponseWriter
}

func (w *panicStatusWriter) Status() int {
	panic("status panic")
}

func (w *panicStatusWriter) Written() bool {
	panic("written panic")
}

func TestSafeWriterStatusAndWrittenRecoverPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	w := &panicStatusWriter{ResponseWriter: c.Writer}

	require.NotPanics(t, func() {
		require.Equal(t, http.StatusOK, safeWriterStatus(w))
		require.False(t, safeWriterWritten(w))
	})
}

func TestLoggerDoesNotPanicWhenWriterStatusPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = initMiddlewareTestLogger(t)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Writer = &panicStatusWriter{ResponseWriter: c.Writer}
		c.Next()
	})
	r.Use(Logger())
	r.GET("/api/test", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	require.NotPanics(t, func() {
		r.ServeHTTP(rec, req)
	})
}

func TestRecoveryDoesNotPanicWhenWriterWrittenPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Writer = &panicStatusWriter{ResponseWriter: c.Writer}
		c.Next()
	})
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	require.NotPanics(t, func() {
		r.ServeHTTP(rec, req)
	})
}
