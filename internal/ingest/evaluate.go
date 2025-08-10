package ingest

import (
	"context"
	"net"
	"regexp"
	"sync"

	"github.com/catonacidd/logship/internal/store"
)

type RulesEngine struct {
	mu    sync.RWMutex
	rules []compiledRule
}
type compiledRule struct {
	name, action string
	re           *regexp.Regexp
}

func NewRulesEngine(db *store.DB) (*RulesEngine, error) {
	rl, err := db.ListRules(context.Background())
	if err != nil { return nil, err }
	re := &RulesEngine{}
	for _, r := range rl {
		cr := compiledRule{name: r.Name, action: r.Action}
		cr.re, _ = regexp.Compile(r.Pattern)
		re.rules = append(re.rules, cr)
	}
	return re, nil
}

func (e *RulesEngine) Add(name, action, pattern string) error {
	e.mu.Lock(); defer e.mu.Unlock()
	rx, err := regexp.Compile(pattern)
	if err != nil { return err }
	e.rules = append(e.rules, compiledRule{name: name, action: action, re: rx})
	return nil
}

func (e *RulesEngine) Evaluate(ev *store.Event) (drop bool, rule string) {
	e.mu.RLock(); defer e.mu.RUnlock()
	for _, r := range e.rules {
		if r.re != nil && r.re.MatchString(ev.Message) {
			return r.action == "drop", r.name
		}
	}
	return false, ""
}

// IP allow/deny

type IPFilter struct {
	wh, bl      []*net.IPNet
	defaultAllow bool
}

func NewIPFilter(lists struct{
	Whitelist []string
	Blacklist []string
	DefaultAllow bool
}) *IPFilter {
	f := &IPFilter{defaultAllow: lists.DefaultAllow}
	for _, c := range lists.Whitelist {
		if _, n, err := net.ParseCIDR(c); err == nil { f.wh = append(f.wh, n) }
	}
	for _, c := range lists.Blacklist {
		if _, n, err := net.ParseCIDR(c); err == nil { f.bl = append(f.bl, n) }
	}
	return f
}

func (f *IPFilter) Allowed(ip string) bool {
	p := net.ParseIP(ip)
	if p == nil { return f.defaultAllow }
	for _, n := range f.bl { if n.Contains(p) { return false } }
	for _, n := range f.wh { if n.Contains(p) { return true } }
	return f.defaultAllow
}
