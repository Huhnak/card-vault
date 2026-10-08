package httpinterface

import (
	"bytes"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	errorPayload bytes.Buffer
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if rw.statusCode >= 400 && rw.errorPayload.Len() < 2048 {
		rw.errorPayload.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		defer func() {
			if rec := recover(); rec != nil {
				duration := time.Since(start)
				slog.Error("http request panic",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Duration("duration", duration),
				)
				http.Error(wrapped, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		fields := []any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", duration),
			slog.String("remote_addr", r.RemoteAddr),
		}

		if wrapped.statusCode >= 400 && wrapped.errorPayload.Len() > 0 {
			fields = append(fields, slog.String("response_body", wrapped.errorPayload.String()))
		}

		switch {
		case wrapped.statusCode >= 500:
			slog.Error("http request server error", fields...)
		case wrapped.statusCode >= 400:
			slog.Warn("http request client error", fields...)
		default:
			slog.Info("http request", fields...)
		}
	})
}
