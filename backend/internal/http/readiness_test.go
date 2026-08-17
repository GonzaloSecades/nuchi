package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GonzaloSecades/nuchi/backend/internal/telemetry"
)

func TestReadinessIsDistinctFromLiveness(t *testing.T) {
	dependencyUp := true
	readiness := NewReadiness(func(context.Context) error {
		if !dependencyUp {
			return errors.New("database unavailable: secret-diagnostic")
		}
		return nil
	})
	router := newRouter(nil, nil, RouterOptions{
		Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), Recorder: telemetry.Noop{}, Readiness: readiness,
	})

	assertStatus := func(path string, want int) string {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != want {
			t.Fatalf("%s: got %d, want %d", path, response.Code, want)
		}
		return response.Body.String()
	}

	assertStatus("/api/ready", http.StatusOK)
	dependencyUp = false
	body := assertStatus("/api/ready", http.StatusServiceUnavailable)
	if strings.Contains(body, "secret-diagnostic") {
		t.Fatal("readiness exposed a dependency diagnostic")
	}
	assertStatus("/api/health", http.StatusOK)

	readiness.SetDraining()
	assertStatus("/api/ready", http.StatusServiceUnavailable)
}

type requestCapture struct {
	result telemetry.RequestResult
}

func (capture *requestCapture) RecordRequest(_ context.Context, result telemetry.RequestResult) {
	capture.result = result
}

func (*requestCapture) RecordDependency(context.Context, telemetry.DependencyResult) {}

func TestRequestObservabilityUsesBoundedRedactedFields(t *testing.T) {
	var logs bytes.Buffer
	capture := &requestCapture{}
	router := newRouter(nil, nil, RouterOptions{
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Recorder: capture, Readiness: NewReadiness(nil),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/health?token=secret-query", strings.NewReader("secret-body"))
	request.Header.Set("Authorization", "Bearer secret-authorization")
	request.Header.Set("Cookie", "refresh_token=secret-cookie")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("response omitted X-Request-ID")
	}
	if capture.result.Operation != "getHealth" || capture.result.Route != "/api/health" || capture.result.StatusClass != "2xx" {
		t.Fatalf("unexpected bounded metrics: %+v", capture.result)
	}
	for _, secret := range []string{"secret-query", "secret-body", "secret-authorization", "secret-cookie"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("request log leaked %q", secret)
		}
	}
}
