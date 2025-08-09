package ingest

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

	syslog "github.com/influxdata/go-syslog/v3"
	rfc5424 "github.com/influxdata/go-syslog/v3/rfc5424"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/rules"
	"github.com/CatOnAcidd/logship/internal/store"
)

func RunSyslog(ctx context.Context, db *store.DB, cfg *config.Config) {
	// TCP
	if cfg.Server.SyslogTCPListen != "" {
		go func() {
			ln, err := net.Listen("tcp", cfg.Server.SyslogTCPListen)
			if err != nil {
				log.Printf("syslog tcp listen error: %v", err)
				return
			}
			log.Printf("syslog TCP listening on %s", cfg.Server.SyslogTCPListen)
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				go handleSyslogConn(ctx, db, conn)
			}
		}()
	}

	// UDP
	if cfg.Server.SyslogUDPListen != "" {
		go func() {
			addr, _ := net.ResolveUDPAddr("udp", cfg.Server.SyslogUDPListen)
			conn, err := net.ListenUDP("udp", addr)
			if err != nil {
				log.Printf("syslog udp listen error: %v", err)
				return
			}
			log.Printf("syslog UDP listening on %s", cfg.Server.SyslogUDPListen)

			buf := make([]byte, 8192)
			machine := rfc5424.NewMachine() // returns syslog.Machine

			for {
				n, remote, err := conn.ReadFromUDP(buf)
				if err != nil {
					return
				}
				line := buf[:n]
				if e := parseAndStoreSyslog(ctx, db, machine, string(line), remote.IP.String()); e != nil {
					log.Printf("syslog udp parse/store: %v", e)
				}
			}
		}()
	}
}

func handleSyslogConn(ctx context.Context, db *store.DB, conn net.Conn) {
	defer conn.Close()
	host := conn.RemoteAddr().String()
	sc := bufio.NewScanner(conn)

	// MVP: line-delimited framing
	machine := rfc5424.NewMachine() // syslog.Machine

	for sc.Scan() {
		line := sc.Text()
		if err := parseAndStoreSyslog(ctx, db, machine, line, host); err != nil {
			log.Printf("syslog tcp parse/store: %v", err)
		}
	}
}

func parseAndStoreSyslog(ctx context.Context, db *store.DB, machine syslog.Machine, msg string, host string) error {
	parsed, err := machine.Parse([]byte(msg))

	obj := map[string]any{
		"message":   msg,
		"parsed_ok": err == nil && parsed != nil,
	}
	if err != nil {
		obj["parse_error"] = err.Error()
	}
	raw, _ := json.Marshal(obj)

	ev := &store.Event{
		Source: "syslog",
		Host:   host,
		Raw:    raw,
		TS:     time.Now().UnixMilli(),
	}

	eng := rules.New(db.SQL())
	_ = eng.Load(ctx)
	return evaluateAndStore(ctx, db, eng, ev)
}
