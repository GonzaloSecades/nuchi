package httpapi

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/google/uuid"
)

func TestEndpointPoliciesCoverOpenAPIExactly(t *testing.T) {
	raw, err := os.ReadFile("../../../openapi/nuchi.openapi.json")
	if err != nil {
		t.Fatalf("read OpenAPI source: %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode OpenAPI source: %v", err)
	}

	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI source has no paths object")
	}
	contractIDs := make([]string, 0, len(endpointPolicies))
	for _, pathValue := range paths {
		pathItem, ok := pathValue.(map[string]any)
		if !ok {
			continue
		}
		for _, operationValue := range pathItem {
			operation, ok := operationValue.(map[string]any)
			if !ok {
				continue
			}
			if operationID, ok := operation["operationId"].(string); ok {
				contractIDs = append(contractIDs, operationID)
			}
		}
	}

	policyIDs := make([]string, 0, len(endpointPolicies))
	for operationID, policy := range endpointPolicies {
		policyIDs = append(policyIDs, operationID)
		if policy.OperationID != operationID {
			t.Errorf("policy key %q carries operation id %q", operationID, policy.OperationID)
		}
		if policy.RequestTimeout <= 0 {
			t.Errorf("policy %q has no request timeout", operationID)
		}
	}

	sort.Strings(contractIDs)
	sort.Strings(policyIDs)
	if got, want := len(policyIDs), len(contractIDs); got != want {
		t.Fatalf("policy count %d does not match contract operation count %d\npolicies: %v\ncontract: %v", got, want, policyIDs, contractIDs)
	}
	for i := range contractIDs {
		if policyIDs[i] != contractIDs[i] {
			t.Fatalf("endpoint policy registry differs from OpenAPI\npolicies: %v\ncontract: %v", policyIDs, contractIDs)
		}
	}
}

func TestEndpointPolicySecurityBoundaries(t *testing.T) {
	cases := []struct {
		operationID    string
		authentication AuthenticationMode
		ownership      OwnershipMode
	}{
		{"getHealth", AuthenticationPublic, OwnershipNone},
		{"loginUser", AuthenticationPublic, OwnershipNone},
		{"refreshSession", AuthenticationRefreshCookie, OwnershipSession},
		{"logoutUser", AuthenticationRefreshCookie, OwnershipSession},
		{"listAccounts", AuthenticationBearer, OwnershipPrincipal},
		{"bulkCreateTransactions", AuthenticationBearer, OwnershipPrincipal},
		{"getSummary", AuthenticationBearer, OwnershipPrincipal},
	}

	for _, tc := range cases {
		t.Run(tc.operationID, func(t *testing.T) {
			policy, ok := EndpointPolicyFor(tc.operationID)
			if !ok {
				t.Fatal("policy missing")
			}
			if policy.Authentication != tc.authentication || policy.Ownership != tc.ownership {
				t.Fatalf("got auth=%q ownership=%q, want auth=%q ownership=%q", policy.Authentication, policy.Ownership, tc.authentication, tc.ownership)
			}
		})
	}
}

func TestRequireAuthStoresTypedPrincipal(t *testing.T) {
	want := Principal{UserID: uuid.MustParse("10000000-0000-4000-8000-000000000001")}
	ctx := context.WithValue(context.Background(), principalContextKey{}, want)

	got, ok := PrincipalFromContext(ctx)
	if !ok {
		t.Fatal("principal missing")
	}
	if got != want {
		t.Fatalf("got principal %+v, want %+v", got, want)
	}
}
