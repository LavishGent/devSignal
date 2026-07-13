// Package httpapi exposes DevSignal over a versioned JSON API and serves the
// built frontend in production.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"devsignal/internal/validation"
)

// maxBodyBytes caps request bodies; career records are small.
const maxBodyBytes = 1 << 20 // 1 MiB

// apiError is the uniform error envelope.
type apiError struct {
	Error  string            `json:"error"`
	Fields validation.Errors `json:"fields,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, apiError{Error: msg})
}

func respondValidation(w http.ResponseWriter, errs validation.Errors) {
	respondJSON(w, http.StatusBadRequest, apiError{Error: "validation failed", Fields: errs})
}

// decodeJSON reads a bounded JSON body into dst, returning a user-facing
// error message when the payload is malformed.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &maxErr):
			return fmt.Errorf("request body must be smaller than %d bytes", maxBodyBytes)
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("request body contains malformed JSON (at offset %d)", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			field := typeErr.Field
			if field == "" {
				field = "body"
			}
			return fmt.Errorf("request body has the wrong type for %q", field)
		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")
		default:
			return errors.New("request body could not be parsed")
		}
	}
	// Reject trailing garbage after the JSON document.
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
