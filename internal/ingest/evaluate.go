package ingest

import (
	"context"
	"net"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
)

func allowedIP(cfg *config.Config, ip string) bool {
	if ip == "" {
		return true
	}
	if _, bad := cfg.Blacklist[ip]; bad {
		return false
	}
	if len(cfg.Whitelist) == 0 {
		return true
	}
	_, ok := cfg.Whitelist[ip]
	return ok
}

func eventFrom(addr net.Addr, host, level, msg string) store.Event {
	ip := ""
	if addr != nil {
		if ua, ok := addr.(*net.UDPAddr); ok && ua.IP != nil {
			ip = ua.IP.String()
		}
		if ta, ok := addr.(*net.TCPAddr); ok && ta.IP != nil {
			ip = ta.IP.String()
		}
	}
	return store.Event{
		Host:     host,
		Level:    level,
		Message:  msg,
		SourceIP: ip,
	}
}

func evaluateAndStore(ctx context.Context, db *store.DB, cfg *config.Config, e store.Event) error {
	if !allowedIP(cfg, e.SourceIP) {
		return db.InsertDrop(ctx, &e)
	}
	return db.InsertEvent(ctx, &e)
}
