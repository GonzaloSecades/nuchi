package httpapi

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTransactionCursorRoundTrip(t *testing.T) {
	wantDate := time.Date(2026, 8, 17, 12, 34, 56, 789, time.UTC)
	wantID := "opaque-resource-id"

	date, id, err := decodeTransactionCursor(encodeTransactionCursor(wantDate, wantID))
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !date.Equal(wantDate) || id != wantID {
		t.Fatalf("got (%s, %q), want (%s, %q)", date, id, wantDate, wantID)
	}
}

func TestParseTransactionPage(t *testing.T) {
	cursor := encodeTransactionCursor(time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC), "txn-1")
	page, err := parseTransactionPage(url.Values{"limit": {"100"}, "cursor": {cursor}})
	if err != nil {
		t.Fatalf("parse valid page: %v", err)
	}
	if page.limit != 100 || !page.queryLimit.Valid || page.queryLimit.Int32 != 101 {
		t.Fatalf("unexpected page limit: %+v", page)
	}
	if !page.cursorDate.Valid || !page.cursorID.Valid || page.cursorID.String != "txn-1" {
		t.Fatalf("unexpected cursor: %+v", page)
	}

	for name, values := range map[string]url.Values{
		"empty limit":          {"limit": {""}},
		"zero limit":           {"limit": {"0"}},
		"too-large limit":      {"limit": {"501"}},
		"non-numeric limit":    {"limit": {"many"}},
		"cursor without limit": {"cursor": {cursor}},
		"invalid cursor":       {"limit": {"10"}, "cursor": {"not-base64"}},
		"oversized cursor":     {"limit": {"10"}, "cursor": {strings.Repeat("a", maxTransactionCursorSize+1)}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTransactionPage(values); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
