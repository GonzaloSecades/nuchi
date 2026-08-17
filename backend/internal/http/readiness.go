package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

const readinessCheckTimeout = 500 * time.Millisecond

// Readiness separates dependency health and graceful drain from liveness.
// The dependency check must honor its context and must not expose diagnostics
// to the public response.
type Readiness struct {
	accepting atomic.Bool
	check     func(context.Context) error
}

func NewReadiness(check func(context.Context) error) *Readiness {
	readiness := &Readiness{check: check}
	readiness.accepting.Store(true)
	return readiness
}

func (r *Readiness) SetDraining() {
	if r != nil {
		r.accepting.Store(false)
	}
}

func (r *Readiness) handler(w http.ResponseWriter, request *http.Request) {
	status := http.StatusOK
	response := readinessResponse{Status: "ready"}
	if r == nil || !r.accepting.Load() {
		status = http.StatusServiceUnavailable
		response.Status = "not_ready"
	} else if r.check != nil {
		ctx, cancel := context.WithTimeout(request.Context(), readinessCheckTimeout)
		defer cancel()
		if err := r.check(ctx); err != nil {
			status = http.StatusServiceUnavailable
			response.Status = "not_ready"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

type readinessResponse struct {
	Status string `json:"status"`
}
