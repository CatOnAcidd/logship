package ingest

import (
	"bufio"
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/catonacidd/logship/internal/store"
)

// RunSyslog starts UDP/TCP listeners if addresses are non-empty (e.g. ":5514").
// It uses a very lightweight parser to avoid external deps for now.
func RunSyslog(ctx context.Context, db *store.DB, udpAddr, tcpAddr string) {
	if udpAddr != "" {
		go runUDP(ctx, db, udpAddr)
	}
	if tcpAddr != "" {
		go runTCP(ctx, db, tcpAddr)
	}
}

func runUDP(ctx context.Context, db *store.DB, addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil { log.Printf("syslog/udp resolve: %v", err); return }
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil { log.Printf("syslog/udp listen: %v", err); return }
	log.Printf("syslog: udp listening on %s", addr)
	defer conn.Close()

	buf := make([]byte, 64*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			log.Printf("syslog/udp read: %v", err)
			continue
		}
		line := strings.TrimSpace(string(buf[:n]))
		lr := parseSyslogLine(line)
		lr.SourceIP = remote.IP.String()
		_ = db.InsertLog(context.Background(), lr)
	}
}

func runTCP(ctx context.Context, db *store.DB, addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil { log.Printf("syslog/tcp listen: %v", err); return }
	log.Printf("syslog: tcp listening on %s", addr)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("syslog/tcp accept: %v", err)
			continue
		}
		go handleTCPConn(ctx, db, conn)
	}
}

func handleTCPConn(ctx context.Context, db *store.DB, c net.Conn) {
	defer c.Close()
	remote := ""
	if ra := c.RemoteAddr(); ra != nil {
		if host, _, err := net.SplitHostPort(ra.String()); err == nil {
			remote = host
		} else {
			remote = ra.String()
		}
	}
	s := bufio.NewScanner(c)
	s.Buffer(make([]byte, 0, 4096), 64*1024)
	for s.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := strings.TrimSpace(s.Text())
		lr := parseSyslogLine(line)
		lr.SourceIP = remote
		_ = db.InsertLog(context.Background(), lr)
	}
}

// parseSyslogLine: naive best-effort pull of host/level/message.
// We do not fully implement RFC3164/5424 here on purpose (keep deps minimal).
func parseSyslogLine(line string) store.LogRow {
	l := store.LogRow{
		TS:      time.Now().Unix(),
		Message: line,
	}
	// Rough split: "<pri>timestamp host level: message" or "host level message"
	rest := line

	// Zap <PRI>
	if strings.HasPrefix(rest, "<") {
		if i := strings.Index(rest, ">"); i > 0 && i < 6 {
			rest = strings.TrimSpace(rest[i+1:])
		}
	}

	parts := strings.Fields(rest)
	if len(parts) >= 2 {
		// Candidate host in first token if it looks hostname-ish
		if isHostLike(parts[0]) {
			l.Host = parts[0]
			rest = strings.TrimSpace(strings.TrimPrefix(rest, parts[0]))
		}
	}

	// level if present like "INFO", "WARN", "ERROR", "debug" etc
	for _, lvl := range []string{"EMERG", "ALERT", "CRIT", "ERR", "ERROR", "WARN", "WARNING", "NOTICE", "INFO", "DEBUG",
		"emerg", "alert", "crit", "err", "error", "warn", "warning", "notice", "info", "debug"} {
		if strings.HasPrefix(rest, lvl) {
			l.Level = lvl
			rest = strings.TrimSpace(strings.TrimPrefix(rest, lvl))
			break
		}
	}

	l.Message = strings.TrimSpace(rest)
	l.Raw = line
	return l
}

func isHostLike(s string) bool {
	// quick heuristic
	if strings.Contains(s, ".") || strings.Contains(s, "-") || strings.Contains(s, "_") {
		return true
	}
	return false
}
