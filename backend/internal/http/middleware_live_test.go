package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GonzaloSecades/nuchi/backend/internal/auth"
	"github.com/GonzaloSecades/nuchi/backend/internal/db"
	dbgen "github.com/GonzaloSecades/nuchi/backend/internal/db/gen"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// accountProbeHandler builds a handler that runs db.WithUserTx bound to the
// request's authenticated user (RequireAuth having already put it on the
// context) and reports, as JSON, whether an account with the given id
// (query param "id") is visible. This is the "minimal test-only protected
// probe route" the #43 briefing calls for: it exercises RequireAuth ->
// UserIDFromContext -> db.WithUserTx -> the #39 RLS policy end-to-end over
// real HTTP, without implementing any of the #44-#48 resource handlers.
func accountProbeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			// Unreachable through RequireAuth, but fail loudly rather than
			// binding a zero UUID if it ever were.
			http.Error(w, "no authenticated user in context", http.StatusInternalServerError)
			return
		}

		accountID := r.URL.Query().Get("id")

		var visible bool
		err := db.WithUserTx(r.Context(), pool, userID, func(q *dbgen.Queries) error {
			_, err := q.GetAccount(r.Context(), dbgen.GetAccountParams{
				ID:     accountID,
				UserID: pgtype.UUID{Bytes: [16]byte(userID), Valid: true},
			})
			if errors.Is(err, pgx.ErrNoRows) {
				visible = false
				return nil
			}
			if err != nil {
				return err
			}
			visible = true
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"visible": visible})
	}
}

func newAccountProbeRouter(secret []byte, pool *pgxpool.Pool) http.Handler {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(RequireAuth(secret))
		r.Get("/probe/account", accountProbeHandler(pool))
	})
	return router
}

// TestRequireAuth_WithUserTx_CrossUserIsolation_LiveDatabase is the full
// stack end-to-end proof for #43: a real HTTP request carrying user A's
// signed access token, through RequireAuth, through UserIDFromContext,
// through db.WithUserTx, against the #39 RLS policy, can see user A's own
// account; the identical request shape carrying user B's token cannot see
// it at all. This is acceptance criterion 7 exercised through the actual
// production request path, complementing the lower-level, predicate-free
// proof in internal/db/rls_test.go.
func TestRequireAuth_WithUserTx_CrossUserIsolation_LiveDatabase(t *testing.T) {
	databaseURL := liveDatabaseURL(t, "RequireAuth + WithUserTx cross-user isolation test")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	t.Cleanup(pool.Close)

	secret := []byte("middleware-live-test-secret-at-least-32-bytes!!")

	userA := createProbeTestUser(ctx, t, pool, "probe-a")
	userB := createProbeTestUser(ctx, t, pool, "probe-b")
	t.Cleanup(func() {
		for _, id := range []uuid.UUID{userA, userB} {
			if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id); err != nil {
				t.Errorf("cleanup: failed to delete probe test user %q: %v", id, err)
			}
		}
	})

	accountID := "probe-account-" + uuid.NewString()
	if err := db.WithUserTx(ctx, pool, userA, func(q *dbgen.Queries) error {
		_, err := q.CreateAccount(ctx, dbgen.CreateAccountParams{
			ID:     accountID,
			Name:   "Probe Account",
			UserID: pgtype.UUID{Bytes: [16]byte(userA), Valid: true},
		})
		return err
	}); err != nil {
		t.Fatalf("seed account for user A: %v", err)
	}

	router := newAccountProbeRouter(secret, pool)

	tokenA, _, err := auth.IssueAccessToken(secret, userA, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken for user A: %v", err)
	}
	tokenB, _, err := auth.IssueAccessToken(secret, userB, 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueAccessToken for user B: %v", err)
	}

	visibleToA := probeAccountVisibility(t, router, tokenA, accountID)
	if !visibleToA {
		t.Error("user A: expected own account to be visible through RequireAuth + WithUserTx, got not visible")
	}

	visibleToB := probeAccountVisibility(t, router, tokenB, accountID)
	if visibleToB {
		t.Error("user B: expected user A's account to be invisible through RequireAuth + WithUserTx, got visible")
	}
}

func probeAccountVisibility(t *testing.T, router http.Handler, token, accountID string) bool {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/probe/account?id="+accountID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("probe request: expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var out struct {
		Visible bool `json:"visible"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode probe response: %v (body: %s)", err, rec.Body.String())
	}
	return out.Visible
}

func createProbeTestUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool, label string) uuid.UUID {
	t.Helper()

	var id string
	email := fmt.Sprintf("%s-%s@example.test", label, uuid.NewString())
	row := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, 'test-hash') RETURNING id
	`, email)
	if err := row.Scan(&id); err != nil {
		t.Fatalf("failed to insert probe test user %q: %v", label, err)
	}
	return uuid.MustParse(id)
}
