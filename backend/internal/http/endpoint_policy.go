package httpapi

import "time"

// AuthenticationMode is the credential boundary for one OpenAPI operation.
type AuthenticationMode string

const (
	AuthenticationPublic        AuthenticationMode = "public"
	AuthenticationBearer        AuthenticationMode = "bearer"
	AuthenticationRefreshCookie AuthenticationMode = "refresh_cookie"
)

// OwnershipMode states where an operation may derive resource ownership.
type OwnershipMode string

const (
	OwnershipNone      OwnershipMode = "none"
	OwnershipSession   OwnershipMode = "session"
	OwnershipPrincipal OwnershipMode = "principal_rls"
)

// IsolationMode records the database snapshot policy for an operation.
type IsolationMode string

const (
	IsolationNone           IsolationMode = "none"
	IsolationReadCommitted  IsolationMode = "read_committed"
	IsolationRepeatableRead IsolationMode = "repeatable_read"
)

// EndpointPolicy is the fail-closed inventory for an OpenAPI operation.
// Body limits owned by Claude tickets remain explicit as noBodyLimit until
// those contract changes land; the registry describes shipped policy, not a
// parallel implementation of #122 or #125.
type EndpointPolicy struct {
	OperationID         string
	Authentication      AuthenticationMode
	Ownership           OwnershipMode
	MaxRequestBodyBytes int64
	RequestTimeout      time.Duration
	Isolation           IsolationMode
	RateLimitClass      string
	Idempotency         string
	SensitiveData       string
}

const defaultEndpointTimeout = 15 * time.Second

func publicPolicy(operationID string, maxBytes int64) EndpointPolicy {
	return EndpointPolicy{
		OperationID:         operationID,
		Authentication:      AuthenticationPublic,
		Ownership:           OwnershipNone,
		MaxRequestBodyBytes: maxBytes,
		RequestTimeout:      defaultEndpointTimeout,
		Isolation:           IsolationNone,
		RateLimitClass:      "none",
		Idempotency:         "command",
		SensitiveData:       "request_body",
	}
}

func refreshCookiePolicy(operationID string) EndpointPolicy {
	policy := publicPolicy(operationID, maxAuthBodyBytes)
	policy.Authentication = AuthenticationRefreshCookie
	policy.Ownership = OwnershipSession
	policy.SensitiveData = "refresh_cookie"
	return policy
}

func financePolicy(operationID string, maxBytes int64, mutation bool) EndpointPolicy {
	policy := EndpointPolicy{
		OperationID:         operationID,
		Authentication:      AuthenticationBearer,
		Ownership:           OwnershipPrincipal,
		MaxRequestBodyBytes: maxBytes,
		RequestTimeout:      defaultEndpointTimeout,
		Isolation:           IsolationReadCommitted,
		RateLimitClass:      "none",
		Idempotency:         "safe_read",
		SensitiveData:       "financial",
	}
	if mutation {
		policy.Idempotency = "non_idempotent_command"
	}
	return policy
}

func transactionPolicy(operationID string, maxBytes int64, mutation bool) EndpointPolicy {
	policy := financePolicy(operationID, maxBytes, mutation)
	if mutation {
		policy.RateLimitClass = "transaction_mutation"
	}
	return policy
}

// endpointPolicies is intentionally keyed by operationId. The contract-sync
// test fails when an operation is added, removed, or renamed without an
// explicit policy decision.
var endpointPolicies = map[string]EndpointPolicy{
	"getHealth": {
		OperationID:         "getHealth",
		Authentication:      AuthenticationPublic,
		Ownership:           OwnershipNone,
		MaxRequestBodyBytes: 0,
		RequestTimeout:      defaultEndpointTimeout,
		Isolation:           IsolationNone,
		RateLimitClass:      "none",
		Idempotency:         "safe_read",
		SensitiveData:       "none",
	},

	"registerUser":         publicPolicy("registerUser", maxAuthBodyBytes),
	"loginUser":            publicPolicy("loginUser", maxAuthBodyBytes),
	"refreshSession":       refreshCookiePolicy("refreshSession"),
	"logoutUser":           refreshCookiePolicy("logoutUser"),
	"verifyEmail":          publicPolicy("verifyEmail", maxAuthBodyBytes),
	"requestPasswordReset": publicPolicy("requestPasswordReset", maxAuthBodyBytes),
	"confirmPasswordReset": publicPolicy("confirmPasswordReset", maxAuthBodyBytes),

	"listAccounts":       financePolicy("listAccounts", 0, false),
	"createAccount":      financePolicy("createAccount", noBodyLimit, true),
	"bulkDeleteAccounts": financePolicy("bulkDeleteAccounts", noBodyLimit, true),
	"getAccount":         financePolicy("getAccount", 0, false),
	"updateAccount":      financePolicy("updateAccount", noBodyLimit, true),
	"deleteAccount":      financePolicy("deleteAccount", 0, true),

	"listCategories":       financePolicy("listCategories", 0, false),
	"createCategory":       financePolicy("createCategory", noBodyLimit, true),
	"bulkDeleteCategories": financePolicy("bulkDeleteCategories", noBodyLimit, true),
	"getCategory":          financePolicy("getCategory", 0, false),
	"updateCategory":       financePolicy("updateCategory", noBodyLimit, true),
	"deleteCategory":       financePolicy("deleteCategory", 0, true),

	"listTransactions":       transactionPolicy("listTransactions", 0, false),
	"createTransaction":      transactionPolicy("createTransaction", noBodyLimit, true),
	"bulkCreateTransactions": transactionPolicy("bulkCreateTransactions", maxBulkCreateBodyBytes, true),
	"bulkDeleteTransactions": transactionPolicy("bulkDeleteTransactions", maxBulkDeleteBodyBytes, true),
	"getTransaction":         transactionPolicy("getTransaction", 0, false),
	"updateTransaction":      transactionPolicy("updateTransaction", noBodyLimit, true),
	"deleteTransaction":      transactionPolicy("deleteTransaction", 0, true),

	"getSummary": func() EndpointPolicy {
		policy := financePolicy("getSummary", 0, false)
		policy.Isolation = IsolationRepeatableRead
		return policy
	}(),
}

// EndpointPolicyFor exposes a read-only copy of the policy for tests and
// future middleware composition. A missing operation fails closed.
func EndpointPolicyFor(operationID string) (EndpointPolicy, bool) {
	policy, ok := endpointPolicies[operationID]
	return policy, ok
}
