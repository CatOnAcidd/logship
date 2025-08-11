package ingest

import (
	"context"
	"strings"

	"github.com/catonacidd/logship/internal/store"
)

func evaluateAndInsert(ctx context.Context, db *store.DB, e *store.Event, rules []store.Rule) error {
	// Base: substring keep/drop; last-match wins, default keep
	decision := "keep"
	for _, r := range rules {
		if r.Kind == "substring" && r.Expr != "" && strings.Contains(strings.ToLower(e.Message), strings.ToLower(r.Expr)) {
			decision = r.Action
		}
	}
	e.Dropped = decision == "drop"
	return db.InsertEvent(ctx, e)
}
