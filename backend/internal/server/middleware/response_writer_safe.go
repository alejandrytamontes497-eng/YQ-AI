package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func safeWriterStatus(w gin.ResponseWriter) (status int) {
	if w == nil {
		return http.StatusOK
	}
	defer func() {
		if recover() != nil {
			status = http.StatusOK
		}
	}()
	return w.Status()
}

func safeWriterWritten(w gin.ResponseWriter) (written bool) {
	if w == nil {
		return false
	}
	defer func() {
		if recover() != nil {
			written = false
		}
	}()
	return w.Written()
}
