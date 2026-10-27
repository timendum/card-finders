package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// methodNotAllowed sets the Allow header and writes a 405 error envelope.
func methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, errorBody{Error: errorDetail{
		Code:    "method_not_allowed",
		Message: "method not allowed",
	}})
}

// decodeJSON decodes the request body into dst, returning false after writing a
// 400 error envelope on malformed input so callers can `if !decodeJSON(...) {
// return }`. Unknown fields, an empty body, and trailing content are all
// rejected.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		code := "bad_request"
		msg := "request body must be valid JSON"
		if errors.Is(err, io.EOF) {
			msg = "request body must not be empty"
		}
		writeError(w, restError{http.StatusBadRequest, code}, errors.New(msg))
		return false
	}
	if dec.More() {
		writeError(w, restError{http.StatusBadRequest, "bad_request"},
			errors.New("request body must contain a single JSON object"))
		return false
	}
	return true
}
