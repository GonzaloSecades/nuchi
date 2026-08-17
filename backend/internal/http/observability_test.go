package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTelemetryOperationIDsMatchOpenAPI(t *testing.T) {
	contractPath := filepath.Join("..", "..", "..", "openapi", "nuchi.openapi.json")
	contents, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var contract struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(contents, &contract); err != nil {
		t.Fatalf("decode contract: %v", err)
	}

	want := make(map[string]bool)
	httpMethods := map[string]bool{"get": true, "post": true, "patch": true, "delete": true, "put": true}
	for _, path := range contract.Paths {
		for method, rawOperation := range path {
			if !httpMethods[method] {
				continue
			}
			var operation struct {
				OperationID string `json:"operationId"`
			}
			if err := json.Unmarshal(rawOperation, &operation); err != nil {
				t.Fatalf("decode %s operation: %v", method, err)
			}
			if operation.OperationID != "" {
				want[operation.OperationID] = true
			}
		}
	}
	got := make(map[string]bool, len(operationIDs))
	for _, operation := range operationIDs {
		got[operation] = true
	}
	for operation := range want {
		if !got[operation] {
			t.Errorf("OpenAPI operation %q has no telemetry mapping", operation)
		}
	}
	for operation := range got {
		if !want[operation] {
			t.Errorf("telemetry operation %q is absent from OpenAPI", operation)
		}
	}
}
