package transform

import "github.com/catonacidd/logship/internal/config"

type Engine struct{
	cfg *config.Config
}

func New(cfg *config.Config) (*Engine, error) { return &Engine{cfg: cfg}, nil }

// Apply returns the input as-is (placeholder for Premium transformations).
func (e *Engine) Apply(m map[string]any) map[string]any { return m }
