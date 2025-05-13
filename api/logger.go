package api

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func HttpLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		bw := &bodyWriter{
			body:           bytes.NewBuffer(nil),
			ResponseWriter: c.Writer,
		}
		c.Writer = bw
		c.Next()

		duration := time.Since(start)
		statusCode := bw.Status()

		logger := log.Info()

		if statusCode >= 400 {
			logger = log.Error().Bytes("body", bw.body.Bytes())
		}

		logger.
			Str("protocol", "http").
			Str("method", c.Request.Method).
			Str("path", c.Request.RequestURI).
			Int("status_code", statusCode).
			Str("status_text", http.StatusText(statusCode)).
			Dur("duration", duration).
			Msg("received an HTTP request")
	}
}
