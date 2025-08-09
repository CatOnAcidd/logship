package transform

import (
	"log"

	"github.com/itchyny/gojq"

	"github.com/CatOnAcidd/logship/internal/config"
)

type Engine struct {
	progs []*gojq.Code
}

func New(cfg *config.Config) (*Engine, error) {
	e := &Engine{}
	for _, r := range cfg.Transforms {
		q, err := gojq.Parse(r.JQ)
		if err != nil {
			return nil, err
		}
		code, err := gojq.Compile(q)
		if err != nil {
			return nil, err
		}
		e.progs = append(e.progs, code)
	}
	return e, nil
}

// Apply runs the pipeline over an input map. Returns the last successful object, or nil.
func (e *Engine) Apply(in map[string]any) map[string]any {
	if e == nil || len(e.progs) == 0 {
		return in
	}
	var cur any = in
	for _, p := range e.progs {
		iter := p.Run(cur)
		v, ok := iter.Next()
		if !ok {
			continue
		}
		if err, isErr := v.(error); isErr {
			log.Printf("transform error: %v", err)
			continue
		}
		cur = v
	}
	out, _ := cur.(map[string]any)
	if out == nil {
		return in
	}
	return out
}
