package ingest

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

	syslog "github.com/influxdata/go-syslog/v3"
	rfc5424 "github.com/influxdata/go-syslog/v3/rfc5424"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/store"
)

func RunSyslog(ctx context.Context, db *store.DB, cfg *config.Config) {
	parser := rfc5424.NewParser()
	// TCP
	if cfg.Server.SyslogTCPListen != "" {
		go func() {
			ln, err := net.Listen("tcp", cfg.Server.SyslogTCPListen)
			if err != nil { log.Printf("syslog tcp listen error: %v", err); return }
			log.Printf("syslog TCP listening on %s", cfg.Server.SyslogTCPListen)
			for {
				conn, err := ln.Accept()
				if err != nil { return }
				go handleSyslogConn(ctx, db, conn, parser)
			}
		}()
	}
	// UDP
	if cfg.Server.SyslogUDPListen != "" {
		go func() {
			addr, _ := net.ResolveUDPAddr("udp", cfg.Server.SyslogUDPListen)
			conn, err := net.ListenUDP("udp", addr)
			if err != nil { log.Printf("syslog udp listen error: %v", err); return }
			log.Printf("syslog UDP listening on %s", cfg.Server.SyslogUDPListen)
			buf := make([]byte, 8192)
			for {
				n, remote, err := conn.ReadFromUDP(buf)
				if err != nil { return }
				msg := string(buf[:n])
				if e := parseAndStoreSyslog(ctx, db, parser, msg, remote.IP.String()); e != nil {
					log.Printf("syslog udp parse/store: %v", e)
				}
			}
		}()
	}
}

func handleSyslogConn(ctx context.Context, db *store.DB, conn net.Conn, parser syslog.Parser) {
	defer conn.Close()
	buf := make([]byte, 8192)
	host := conn.RemoteAddr().String()
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		msg := string(buf[:n])
		if e := parseAndStoreSyslog(ctx, db, parser, msg, host); e != nil {
			log.Printf("syslog tcp parse/store: %v", e)
		}
	}
}

func parseAndStoreSyslog(ctx context.Context, db *store.DB, parser syslog.Parser, msg string, host string) error {
	m, err := parser.Parse(msg)
	if err != nil {
		return err
	}
	obj := map[string]any{
		"message": msg,
		"parsed":  m,
	}
	raw, _ := json.Marshal(obj)
	ev := &store.Event{
		Source: "syslog",
		Host:   host,
		Level:  "", // could map from m.Severity
		Raw:    raw,
		TS:     time.Now().UnixMilli(),
	}
	return db.Insert(ctx, ev)
}
