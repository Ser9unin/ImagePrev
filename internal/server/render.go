package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// JSON sends json response
func responseJSON(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
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

// JSONMap is a map alias
type JSONMap map[string]interface{}

// ErrorJSON sends error as json
func ErrorJSON(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string) {
	responseJSON(w, r, httpStatusCode, JSONMap{"error": err.Error(), "details": details})
}

// NoContent sends no content response
func NoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

var (
	ErrNotFound            = errors.New("your requested item is not found")
	ErrInternalServerError = errors.New("internal server error")
)

// StatusCode gets http code from error
func StatusCode(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}

	return http.StatusInternalServerError
}
