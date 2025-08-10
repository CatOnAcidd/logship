package ingest

import (
	"bufio"
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/CatOnAcidd/logship/internal/store"
)

func parseLine(addr, line string) store.Event {
	line = strings.TrimSpace(line)
	lvl := "info"
	host := ""
	msg := line
	// very naive parse for "<PRI>timestamp host level message"
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		host = parts[0]
		msg = strings.Join(parts[1:], " ")
	}
	return store.Event{Host: host, Level: lvl, Message: msg, SourceIP: addr, ReceivedAt: store.Now()}
}

func RunSyslogUDP(ctx context.Context, db *store.DB, addr string) {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil { log.Printf("syslog udp listen %s: %v", addr, err); return }
	log.Printf("syslog udp listening on %s", addr)
	buf := make([]byte, 64*1024)
	for {
		select { case <-ctx.Done(): pc.Close(); return default: }
		n, raddr, err := pc.ReadFrom(buf)
		if err != nil { continue }
		line := string(buf[:n])
		ev := parseLine(addrString(raddr), line)
		_ = db.Insert(ctx, ev)
	}
}

func RunSyslogTCP(ctx context.Context, db *store.DB, addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil { log.Printf("syslog tcp listen %s: %v", addr, err); return }
	log.Printf("syslog tcp listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil { continue }
		go func(c net.Conn) {
			defer c.Close()
			s := bufio.NewScanner(c)
			s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			peer := addrString(c.RemoteAddr())
			for s.Scan() {
				line := s.Text()
				ev := parseLine(peer, line)
				_ = db.Insert(ctx, ev)
			}
		}(conn)
	}
}

func addrString(a net.Addr) string {
	if a == nil { return "" }
	s := a.String()
	if i := strings.LastIndex(s, ":"); i > 0 { return s[:i] }
	return s
}

var _ = time.Now // quiet import
