// Package api exposes the math operations as an HTTP API.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/betodalas/teste-sre-roberto/internal/mathops"
)

// resultResponse is the JSON body returned by every operation endpoint.
type resultResponse struct {
	Result int64 `json:"result"`
}

// errorResponse is the JSON body returned when a request cannot be
// processed.
type errorResponse struct {
	Error string `json:"error"`
}

// NewRouter builds the HTTP handler exposing the four arithmetic
// operations plus a healthcheck endpoint.
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/sum", operationHandler(func(a, b int64) (int64, error) {
		return mathops.Sum(a, b), nil
	}))
	mux.HandleFunc("/api/sub", operationHandler(func(a, b int64) (int64, error) {
		return mathops.Sub(a, b), nil
	}))
	mux.HandleFunc("/api/mul", operationHandler(func(a, b int64) (int64, error) {
		return mathops.Mul(a, b), nil
	}))
	mux.HandleFunc("/api/div", operationHandler(mathops.Div))

	mux.HandleFunc("/healthz", healthzHandler)

	return mux
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// operationHandler adapts a two-argument arithmetic operation into an
// http.HandlerFunc that reads term_one and term_two from the query string.
func operationHandler(op func(termOne, termTwo int64) (int64, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		termOne, err := parseIntParam(r, "term_one")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		termTwo, err := parseIntParam(r, "term_two")
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := op(termOne, termTwo)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, resultResponse{Result: result})
	}
}

func parseIntParam(r *http.Request, name string) (int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, missingParamError(name)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, invalidParamError(name)
	}
	return value, nil
}

func missingParamError(name string) error {
	return &paramError{msg: "missing query parameter: " + name}
}

func invalidParamError(name string) error {
	return &paramError{msg: "invalid integer value for query parameter: " + name}
}

type paramError struct{ msg string }

func (e *paramError) Error() string { return e.msg }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
