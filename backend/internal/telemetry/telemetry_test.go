package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestNoopRecorder(t *testing.T) {
	recorder := Recorder(Noop{})
	recorder.RecordRequest(context.Background(), RequestResult{
		Operation:   "listTransactions",
		Method:      "GET",
		Route:       "/api/transactions/",
		StatusClass: "2xx",
		Duration:    time.Millisecond,
	})
	recorder.RecordDependency(context.Background(), DependencyResult{
		Kind: "database", Name: "ListTransactions", Outcome: "ok", Duration: time.Millisecond,
	})
}
