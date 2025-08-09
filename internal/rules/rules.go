package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionDrop  Action = "drop"
)

type PredicateType string

const (
	PredRegex PredicateType = "regex"
	// PredJQ reserved for premium
)

type Predicate struct {
	Type  PredicateType `json:"type"`
	Field string        `json:"field"` // message|host|app|level|source
	Expr  string        `json:"expr"`  // regex
}

type Rule struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Enabled  bool     `json:"enabled"`
	Priority int      `json:"priority"`
	Source   string   `json:"source"`   // "syslog"|"windows"|"http"|"file"|""(any)
	Action   Action   `json:"action"`   // allow|drop
	PredRaw  string   `json:"predicate"`// JSON blob
	// compiled
	pred Predicate
	re   *regexp.Regexp
}

type EventProbe struct {
	TS      int64          `json:"ts"`
	Source  string         `json:"source"`
	Host    string         `json:"host,omitempty"`
	App     string         `json:"app,omitempty"`
	Level   string         `json:"level,omitempty"`
	Message string         `json:"message,omitempty"`
	Body    map[string]any `json:"body,omitempty"`
	Raw     string         `json:"raw,omitempty"`
}

// Engine holds compiled rules
type Engine struct {
	db    *sql.DB
	rules []Rule
}

func New(db *sql.DB) *Engine { return &Engine{db: db} }

func (e *Engine) Load(ctx context.Context) error {
	rows, err := e.db.QueryContext(ctx, `SELECT id,name,enabled,priority,source,action,predicate FROM rules WHERE enabled=1`)
	if err != nil { return err }
	defer rows.Close()
	var list []Rule
	for rows.Next() {
		var r Rule
		var enabled int
		if err := rows.Scan(&r.ID,&r.Name,&enabled,&r.Priority,&r.Source,&r.Action,&r.PredRaw); err != nil { return err }
		r.Enabled = enabled == 1
		var p Predicate
		if err := json.Unmarshal([]byte(r.PredRaw), &p); err == nil {
			r.pred = p
			if strings.EqualFold(string(p.Type), string(PredRegex)) && p.Expr != "" {
				if re, err := regexp.Compile(p.Expr); err == nil {
					r.re = re
				}
			}
		}
		list = append(list, r)
	}
	sort.Slice(list, func(i,j int) bool { return list[i].Priority < list[j].Priority })
	e.rules = list
	return rows.Err()
}

type Decision struct {
	Action   Action
	RuleID   int64
	RuleName string
	Matched  bool
}

func fieldValue(ev EventProbe, field string) string {
	switch field {
	case "message":
		return ev.Message
	case "host":
		return ev.Host
	case "app":
		return ev.App
	case "level":
		return ev.Level
	case "source":
		return ev.Source
	default:
		if ev.Body != nil {
			if v, ok := ev.Body[field]; ok {
				if s, ok := v.(string); ok { return s }
			}
		}
	}
	return ""
}

// Evaluate returns the first matching rule's decision. If none, default allow.
func (e *Engine) Evaluate(ev EventProbe) Decision {
	for _, r := range e.rules {
		if r.Source != "" && r.Source != ev.Source { continue }
		switch r.pred.Type {
		case PredRegex:
			val := fieldValue(ev, r.pred.Field)
			if r.re != nil && r.re.MatchString(val) {
				return Decision{Action: r.Action, RuleID: r.ID, RuleName: r.Name, Matched: true}
			}
		default:
			continue
		}
	}
	return Decision{Action: ActionAllow, Matched: false}
}

// TestAgainstSamples returns indices of samples that would match this predicate.
func (e *Engine) TestAgainstSamples(pred Predicate, source string, samples []EventProbe, limit int) ([]int, error) {
	if strings.EqualFold(string(pred.Type), string(PredRegex)) {
		re, err := regexp.Compile(pred.Expr)
		if err != nil { return nil, err }
		out := []int{}
		for i, s := range samples {
			if source!="" && source!=s.Source { continue }
			val := fieldValue(s, pred.Field)
			if re.MatchString(val) {
				out = append(out, i)
				if limit>0 && len(out)>=limit { break }
			}
		}
		return out, nil
	}
	return []int{}, nil
}
