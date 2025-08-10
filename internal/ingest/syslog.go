package ingest

import (
	"bytes"
	"context"
	"log"
	"net"
	"time"

	"github.com/catonacidd/logship/internal/store"
	"github.com/influxdata/go-syslog/v3"
	"github.com/influxdata/go-syslog/v3/rfc3164"
	"github.com/influxdata/go-syslog/v3/rfc5424"
)

type SyslogServer struct {
	DB        *store.DB
	Engine    *RulesEngine
	IPFilter  *IPFilter
	UDPListen string
	TCPListen string

	udpConn net.PacketConn
	tcpLn   net.Listener
}

func (s *SyslogServer) Start() {
	if s.UDPListen != "" { go s.runUDP(s.UDPListen) }
	if s.TCPListen != "" { go s.runTCP(s.TCPListen) }
}
func (s *SyslogServer) Shutdown() {
	if s.udpConn != nil { _ = s.udpConn.Close() }
	if s.tcpLn != nil { _ = s.tcpLn.Close() }
}

func (s *SyslogServer) runUDP(addr string) {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil { log.Printf("udp listen %s: %v", addr, err); return }
	s.udpConn = pc
	buf := make([]byte, 64*1024)
	p3164 := rfc3164.NewParser()
	p5424 := rfc5424.NewParser()
	for {
		n, from, err := pc.ReadFrom(buf)
		if err != nil { return }
		src := ipOnly(from.String())
		if s.IPFilter != nil && !s.IPFilter.Allowed(src) { continue }
		msg := parseSyslog(buf[:n], p3164, p5424)
		ev := &store.Event{
			TS:       time.Now().UTC(),
			Host:     msgHostname(msg),
			Level:    msgSeverity(msg),
			Message:  msgMessage(msg),
			SourceIP: src,
		}
		drop, rule := s.Engine.Evaluate(ev)
		if drop { _ = s.DB.InsertDrop(context.Background(), ev, rule) } else { _ = s.DB.Insert(context.Background(), ev) }
	}
}

func (s *SyslogServer) runTCP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil { log.Printf("tcp listen %s: %v", addr, err); return }
	s.tcpLn = ln
	p3164 := rfc3164.NewParser()
	p5424 := rfc5424.NewParser()
	for {
		c, err := ln.Accept()
		if err != nil { return }
		go func(conn net.Conn) {
			defer conn.Close()
			src := ipOnly(conn.RemoteAddr().String())
			if s.IPFilter != nil && !s.IPFilter.Allowed(src) { return }
			buf := make([]byte, 64*1024)
			for {
				n, err := conn.Read(buf)
				if n > 0 {
					msg := parseSyslog(buf[:n], p3164, p5424)
					ev := &store.Event{
						TS:       time.Now().UTC(),
						Host:     msgHostname(msg),
						Level:    msgSeverity(msg),
						Message:  msgMessage(msg),
						SourceIP: src,
					}
					drop, rule := s.Engine.Evaluate(ev)
					if drop { _ = s.DB.InsertDrop(context.Background(), ev, rule) } else { _ = s.DB.Insert(context.Background(), ev) }
				}
				if err != nil { return }
			}
		}(c)
	}
}

func parseSyslog(b []byte, p3164, p5424 syslog.Parser) syslog.Message {
	if m, err := p5424.Parse(bytes.NewReader(b)); err == nil && m != nil { return m }
	if m, err := p3164.Parse(bytes.NewReader(b)); err == nil && m != nil { return m }
	return nil
}
func msgHostname(m syslog.Message) string { if m==nil {return ""}; if v:=m.Hostname(); v!=nil {return *v}; return "" }
func msgSeverity(m syslog.Message) string { if m==nil {return ""}; if v:=m.Severity(); v!=nil {return v.String()}; return "" }
func msgMessage(m syslog.Message) string  { if m==nil {return ""}; if v:=m.Message(); v!=nil {return *v}; return "" }

func ipOnly(s string) string {
	for i := len(s)-1; i >= 0; i-- {
		if s[i] == ':' { return s[:i] }
	}
	return s
}
