package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// statusWriter captures the status code without buffering the body.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// AccessLog emits one structured line per request.
//
// Production-grade log: method, path, status, duration in ms, bytes written,
// request id, and remote addr. No body logging. PII scrubbing happens upstream
// in handlers if needed (this middleware is intentionally oblivious to payload).
func AccessLog(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)

			duration := time.Since(start)

			evt := logger.Info()
			if sw.status >= 500 {
				evt = logger.Error()
			} else if sw.status >= 400 {
				evt = logger.Warn()
			}

			evt.
				Str("request_id", RequestIDFrom(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", sw.status).
				Int("bytes", sw.bytes).
				Dur("duration", duration).
				Str("remote", r.RemoteAddr).
				Msg("http")
		})
	}
}
