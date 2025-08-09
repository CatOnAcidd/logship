package transform

type Engine struct{}

func New() *Engine { return &Engine{} }
func (e *Engine) Apply(m map[string]any) map[string]any { return m }
