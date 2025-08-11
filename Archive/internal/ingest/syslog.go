package ingest

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
)

type SyslogIngest struct {
	db   *store.DB
	cfg  *config.Config
	udp  *net.UDPConn
	tcp  net.Listener
	stop chan struct{}
}

func NewSyslogIngest(db *store.DB, cfg *config.Config) *SyslogIngest {
	return &SyslogIngest{db: db, cfg: cfg, stop: make(chan struct{})}
}

func (s *SyslogIngest) Start() error {
	// UDP
	udpAddr, err := net.ResolveUDPAddr("udp", s.cfg.SyslogUDP)
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	s.udp = udpConn
	go s.serveUDP()

	// TCP
	ln, err := net.Listen("tcp", s.cfg.SyslogTCP)
	if err != nil {
		return err
	}
	s.tcp = ln
	go s.serveTCP()

	log.Printf("syslog: UDP %s, TCP %s", s.cfg.SyslogUDP, s.cfg.SyslogTCP)
	return nil
}

func (s *SyslogIngest) Close() error {
	close(s.stop)
	if s.udp != nil {
		_ = s.udp.Close()
	}
	if s.tcp != nil {
		_ = s.tcp.Close()
	}
	return nil
}

func parseSyslogLine(line []byte) (host, level, msg string) {
	// try 5424, then 3164
	var m syslog.Message
	var err error

	if p := rfc5424.NewParser(); p != nil {
		m, err = p.Parse(bytes.NewReader(line))
		if err == nil && m != nil {
			if h := m.Hostname(); h != nil {
				host = *h
			}
			if l := m.SeverityLevel(); l != nil {
				level = l.String()
			}
			if mm := m.Message(); mm != nil {
				msg = string(*mm)
			}
			if msg == "" {
				msg = string(line)
			}
			return
		}
	}
	if p := rfc3164.NewParser(); p != nil {
		m, err = p.Parse(bytes.NewReader(line))
		if err == nil && m != nil {
			if h := m.Hostname(); h != nil {
				host = *h
			}
			if l := m.Severity(); l != nil {
				level = l.String()
			}
			if mm := m.Message(); mm != nil {
				msg = *mm
			}
			if msg == "" {
				msg = string(line)
			}
			return
		}
	}

	// Fallback
	return "", "", string(line)
}

func (s *SyslogIngest) serveUDP() {
	buf := make([]byte, 64*1024)
	for {
		n, addr, err := s.udp.ReadFromUDP(buf)
		if err != nil {
			return
		}
		host, level, msg := parseSyslogLine(buf[:n])
		if host == "" {
			host = addr.IP.String()
		}
		e := &store.Event{
			TS:      time.Now().UTC(),
			Host:    host,
			Level:   level,
			Message: msg,
		}
		rules, _ := s.db.ListRules(context.Background())
		_ = evaluateAndInsert(context.Background(), s.db, e, rules)
	}
}

func (s *SyslogIngest) serveTCP() {
	for {
		conn, err := s.tcp.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 64*1024)
			for {
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				host, level, msg := parseSyslogLine(buf[:n])
				if host == "" {
					host = remoteHost(c.RemoteAddr())
				}
				e := &store.Event{
					TS:      time.Now().UTC(),
					Host:    host,
					Level:   level,
					Message: msg,
				}
				rules, _ := s.db.ListRules(context.Background())
				_ = evaluateAndInsert(context.Background(), s.db, e, rules)
			}
		}(conn)
	}
}

func remoteHost(a net.Addr) string {
	if a == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(a.String())
	if err != nil {
		return a.String()
	}
	return host
}

// Optional simple file tail
func RunFileTail(db *store.DB, path string, isGlob bool) {
	// Minimal: for now just log a message. (You can expand to real tailing later.)
	fmt.Printf("filetail: configured path=%s glob=%v (not active in base sample)\n", path, isGlob)
}
