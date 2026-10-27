package server

import (
	"cardfinders/internal/engine"
	"encoding/json"
	"errors"
	"net/http"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// restError is the wire form (HTTP status + machine code) an engine error maps to.
type restError struct {
	status int
	code   string
}

var restErrorMap = map[error]restError{
	engine.ErrLevelRequired: {http.StatusBadRequest, "level_required"},
	engine.ErrLevelInvalid:  {http.StatusBadRequest, "level_invalid"},
	engine.ErrGameExists:    {http.StatusConflict, "game_exists"},
	engine.ErrGameFull:      {http.StatusConflict, "game_full"},
	engine.ErrCodeRequired:  {http.StatusBadRequest, "code_required"},
	engine.ErrCodeInvalid:   {http.StatusNotFound, "code_invalid"},
	engine.ErrNoGame:        {http.StatusNotFound, "code_invalid"},
}

// classifyCreateError maps a CreateGame error to its wire form; any unmapped
// error is a ValidateLevel failure surfaced as 400 level_config_invalid.
func classifyCreateError(err error) restError {
	if re, ok := lookupRESTError(err); ok {
		return re
	}
	return restError{http.StatusBadRequest, "level_config_invalid"}
}

// classifyJoinError maps a Join error to its wire form; an unmapped error falls
// back to 400 bad_request so a future engine error never escapes as a 500.
func classifyJoinError(err error) restError {
	if re, ok := lookupRESTError(err); ok {
		return re
	}
	return restError{http.StatusBadRequest, "bad_request"}
}

// lookupRESTError resolves err against restErrorMap via errors.Is so wrapped
// sentinels still match.
func lookupRESTError(err error) (restError, bool) {
	for sentinel, re := range restErrorMap {
		if errors.Is(err, sentinel) {
			return re, true
		}
	}
	return restError{}, false
}

// writeError writes the JSON error envelope, using err's message as the text.
func writeError(w http.ResponseWriter, re restError, err error) {
	writeJSON(w, re.status, errorBody{Error: errorDetail{
		Code:    re.code,
		Message: err.Error(),
	}})
}

// writeJSON serializes v as JSON with the given status. A post-header encode
// error cannot reach the client and is dropped.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
