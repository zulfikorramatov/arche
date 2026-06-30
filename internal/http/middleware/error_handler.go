package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type errorResponseWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
	isErr  bool
}

func (w *errorResponseWriter) WriteHeader(code int) {
	w.status = code
	if code >= 400 {
		w.isErr = true
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *errorResponseWriter) Write(p []byte) (int, error) {
	if w.isErr {
		return w.buf.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func (w *errorResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *errorResponseWriter) flush() {
	if !w.isErr {
		return
	}

	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(w.buf.Bytes())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.ResponseWriter.WriteHeader(w.status)
	_ = json.NewEncoder(w.ResponseWriter).Encode(map[string]string{
		"error": strings.TrimSpace(w.buf.String()),
	})
}

func ErrorHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ew := &errorResponseWriter{ResponseWriter: w}
			next.ServeHTTP(ew, r)
			ew.flush()
		})
	}
}
