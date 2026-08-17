// Package telemetry defines Nuchi's vendor-neutral instrumentation boundary.
// Domain and HTTP code depend on this small contract rather than an exporter
// SDK, so selecting a telemetry backend does not require business-logic edits.
package telemetry

import (
	"context"
	"time"
)

// RequestResult is the bounded, non-sensitive request-completion vocabulary.
// Operation and Route must be stable names; callers must never put raw URLs,
// resource IDs, user IDs, or request IDs in metric dimensions.
type RequestResult struct {
	Operation   string
	Method      string
	Route       string
	StatusClass string
	Duration    time.Duration
}

// DependencyResult describes one named dependency boundary. Name is a stable
// query or integration name, not raw SQL, an address, or a payload value.
type DependencyResult struct {
	Kind     string
	Name     string
	Outcome  string
	Duration time.Duration
}

// Recorder is the adapter seam required by the HTTP, service, transaction,
// database, and external-dependency boundaries. Implementations must be safe
// for concurrent use and must not block request completion on exporter I/O.
type Recorder interface {
	RecordRequest(context.Context, RequestResult)
	RecordDependency(context.Context, DependencyResult)
}

// Noop is the default local/test recorder and deliberately allocates no state.
type Noop struct{}

func (Noop) RecordRequest(context.Context, RequestResult)       {}
func (Noop) RecordDependency(context.Context, DependencyResult) {}

var _ Recorder = Noop{}
