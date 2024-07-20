package server

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// statusWriter is custom http.ResponseWriter that captures status and size of response
type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func HttpLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := statusWriter{ResponseWriter: w}

		next(&sw, r)

		fields := []zapcore.Field{
			zap.Int("status", sw.status),
			zap.Int("response_size", sw.size),
			zap.String("latency", time.Since(start).String()),
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.String("host", r.Host),
			zap.String("remote_ip", r.RemoteAddr),
		}

		n := sw.status
		switch {
		case n >= 500:
			zap.L().Error("Server error", fields...)
		case n >= 400:
			zap.L().Warn("Client error", fields...)
		case n >= 300:
			zap.L().Info("Redirection", fields...)
		default:
			zap.L().Info("Success", fields...)
		}
	}
}

func CheckHttpRequest(client *http.Client, request string) ([]byte, error) {
	req, err := http.NewRequest("GET", request, nil)
	if err != nil {
		err = fmt.Errorf("запрос не сформирован: %w", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("запрос не отправлен: %w", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("не прочитано тело запроса: %w", err)
		return nil, err
	}
	return body, nil
}

func CheckHttpMethod(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			//api.ErrorJSON(w, r, http.StatusBadRequest, fmt.Errorf("bad method: %s", r.Method), "method should be get")
			return
		}
	}
}
