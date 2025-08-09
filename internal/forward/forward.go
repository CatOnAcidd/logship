package forward

import (
	"context"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
	"github.com/catonacidd/logship/internal/transform"
)

type Forwarder struct{
	cfg *config.Config
	db  *store.DB
	tr  *transform.Engine
}

// New returns a no-op forwarder in Base edition.
func New(db *store.DB, cfg *config.Config, tr *transform.Engine) *Forwarder {
	return &Forwarder{cfg: cfg, db: db, tr: tr}
}

// Run is expected by main.go. For Base, it's equivalent to Start(ctx).
func (f *Forwarder) Run(ctx context.Context) error {
	return f.Start(ctx)
}

// Start is a placeholder for Base; Premium can implement multiple destinations.
func (f *Forwarder) Start(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
