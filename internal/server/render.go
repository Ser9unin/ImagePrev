package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// responseImage отправляет клиенту изображение в []byte.
func responseImage(w http.ResponseWriter, _ *http.Request, status int, data []byte) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

// responseJSON отправляет ответ в формате json.
func responseJSON(w http.ResponseWriter, _ *http.Request, status int, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(status)

	_, err := w.Write(buf.Bytes())
	if err != nil {
		log.Println(err)
	}
}

// JSONMap is a map alias.
type JSONMap map[string]interface{}

// ErrorJSON отправляет ошибку в формате json.
func ErrorJSON(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string) {
	responseJSON(w, r, httpStatusCode, JSONMap{"error": err.Error(), "details": details})
}

// NoContent отправляет ответ что контента нет
func NoContent(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

var (
	ErrNotFound            = errors.New("your requested item is not found")
	ErrInternalServerError = errors.New("internal server error")
)

// StatusCode получает http статус из ошибки
func StatusCode(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}

	return http.StatusInternalServerError
}
