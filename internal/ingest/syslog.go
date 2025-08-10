package ingest

import (
	"bufio"
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
)

// Minimal tolerant parser: try "<PRI>ts host msg" or default host="", message=whole line
func parseLine(line string) (host, level, msg string) {
	msg = strings.TrimSpace(line)
	level = "info"
	// crude host extraction if "host: message"
	if i := strings.Index(msg, ": "); i > 0 && i < 64 {
		host = msg[:i]
		msg = strings.TrimSpace(msg[i+2:])
	}
	return
}

func RunSyslogUDP(ctx context.Context, addr string, db *store.DB, cfg *config.Config) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	log.Printf("syslog udp: %s", addr)
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	buf := make([]byte, 64*1024)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, raddr, err := conn.ReadFrom(buf)
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("syslog udp read: %v", err)
			continue
		}
		host, level, msg := parseLine(string(buf[:n]))
		e := eventFrom(raddr, host, level, msg)
		if err := evaluateAndStore(context.Background(), db, cfg, e); err != nil {
			log.Printf("udp store: %v", err)
		}
	}
}

func RunSyslogTCP(ctx context.Context, addr string, db *store.DB, cfg *config.Config) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("syslog tcp: %s", addr)
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("syslog tcp accept: %v", err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			sc := bufio.NewScanner(conn)
			sc.Buffer(make([]byte, 4096), 1024*1024)
			for sc.Scan() {
				line := sc.Text()
				host, level, msg := parseLine(line)
				e := eventFrom(conn.RemoteAddr(), host, level, msg)
				if err := evaluateAndStore(context.Background(), db, cfg, e); err != nil {
					log.Printf("tcp store: %v", err)
				}
			}
		}(c)
	}
}
