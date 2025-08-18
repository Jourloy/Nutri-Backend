package middlewares

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: `[rqst]`,
		Level:  log.DebugLevel,
	})
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		ww := NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		latency := time.Since(t)
		status := ww.Status()
		method := r.Method
		path := r.URL.Path

		logger.Info(
			"Response",
			"method", method,
			"path", path,
			"status", status,
			"latency", latency,
		)

		// If request latency is over limit
		if time.Duration(latency.Milliseconds()) > 500 {
			logger.Warn("Latency is over 500ms")
		}
	})
}

type WrapResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) *WrapResponseWriter {
	return &WrapResponseWriter{ResponseWriter: w}
}

func (w *WrapResponseWriter) Status() int {
	return w.status
}

func (w *WrapResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
		w.wroteHeader = true
	}
}
