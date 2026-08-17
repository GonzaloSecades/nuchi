package httpapi

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxTransactionPageSize   = 500
	maxTransactionCursorSize = 512
)

var (
	errInvalidTransactionLimit  = errors.New("limit must be an integer between 1 and 500.")
	errInvalidTransactionCursor = errors.New("cursor is invalid.")
	errCursorRequiresLimit      = errors.New("cursor requires limit.")
)

type transactionPage struct {
	limit      int
	queryLimit pgtype.Int4
	cursorDate pgtype.Timestamp
	cursorID   pgtype.Text
}

func parseTransactionPage(query url.Values) (transactionPage, error) {
	limitValue := optionalQueryParam(query, "limit")
	cursorValue := optionalQueryParam(query, "cursor")
	if limitValue == nil {
		if cursorValue != nil {
			return transactionPage{}, errCursorRequiresLimit
		}
		return transactionPage{}, nil
	}

	limit, err := strconv.Atoi(*limitValue)
	if err != nil || limit < 1 || limit > maxTransactionPageSize {
		return transactionPage{}, errInvalidTransactionLimit
	}
	page := transactionPage{
		limit:      limit,
		queryLimit: pgtype.Int4{Int32: int32(limit + 1), Valid: true},
	}
	if cursorValue == nil {
		return page, nil
	}
	if len(*cursorValue) == 0 || len(*cursorValue) > maxTransactionCursorSize {
		return transactionPage{}, errInvalidTransactionCursor
	}

	date, id, err := decodeTransactionCursor(*cursorValue)
	if err != nil {
		return transactionPage{}, errInvalidTransactionCursor
	}
	page.cursorDate = pgtype.Timestamp{Time: date, Valid: true}
	page.cursorID = pgtype.Text{String: id, Valid: true}
	return page, nil
}

func encodeTransactionCursor(date time.Time, id string) string {
	payload := date.UTC().Format(time.RFC3339Nano) + "\n" + id
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeTransactionCursor(cursor string) (time.Time, string, error) {
	payload, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(payload), "\n", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", errInvalidTransactionCursor
	}
	date, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return date, parts[1], nil
}
