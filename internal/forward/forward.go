package forward

import (
	"context"
	"github.com/catonacidd/logship/internal/config"
)

type Forwarder struct{
	cfg *config.Config
}

func New(cfg *config.Config) (*Forwarder, error) {
	return &Forwarder{cfg: cfg}, nil
}

// Start is a placeholder for Base; Premium can implement multiple destinations.
func (f *Forwarder) Start(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
