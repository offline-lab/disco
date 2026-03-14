package dns

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type RecordProvider interface {
	GetAllRecords() []DNSRecord
}

type DNSRecord struct {
	Hostname  string
	Addresses []string
	Services  map[string]ServiceInfo
	Status    string
	LastSeen  time.Time
	IsStatic  bool
}

type ServiceInfo struct {
	Port     int
	Protocol string
}

type Server struct {
	config   *Config
	provider RecordProvider
	server   *dns.Server
	stopChan chan struct{}
}

type Config struct {
	Enabled       bool
	Port          int
	Domain        string
	BindAddresses []string
	TTLHealthy    int
	TTLStale      int
}

func NewServer(cfg *Config, provider RecordProvider) *Server {
	return &Server{
		config:   cfg,
		provider: provider,
		stopChan: make(chan struct{}),
	}
}

func (s *Server) Start() error {
	if !s.config.Enabled {
		return nil
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(s.config.Domain+".", s.handleQuery)
	mux.HandleFunc("in-addr.arpa.", s.handleReverse)

	for _, addr := range s.config.BindAddresses {
		bindAddr := fmt.Sprintf("%s:%d", addr, s.config.Port)

		server := &dns.Server{
			Addr:    bindAddr,
			Net:     "udp",
			Handler: mux,
		}

		go func() {
			if err := server.ListenAndServe(); err != nil {
				fmt.Printf("DNS server error on %s: %v\n", bindAddr, err)
			}
		}()
	}

	return nil
}

func (s *Server) Stop() {
	close(s.stopChan)
	if s.server != nil {
		s.server.Shutdown()
	}
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	domain := s.config.Domain + "."

	for _, q := range r.Question {
		switch q.Qtype {
		case dns.TypeA:
			s.handleAQuery(q, m, domain)
		case dns.TypeAAAA:
			// No AAAA records for now
		case dns.TypeTXT:
			s.handleTXTQuery(q, m, domain)
		case dns.TypeSRV:
			s.handleSRVQuery(q, m, domain)
		case dns.TypeCNAME:
			s.handleCNAMEQuery(q, m, domain)
		}
	}

	w.WriteMsg(m)
}

func (s *Server) handleReverse(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype == dns.TypePTR {
			s.handlePTRQuery(q, m)
		}
	}

	w.WriteMsg(m)
}

func (s *Server) handleAQuery(q dns.Question, m *dns.Msg, domain string) {
	name := strings.TrimSuffix(q.Name, ".")
	name = strings.TrimSuffix(name, domain)

	if name == "" {
		return
	}

	records := s.provider.GetAllRecords()
	for _, rec := range records {
		if rec.Hostname == name {
			ttl := s.getTTL(rec.Status)
			for _, addr := range rec.Addresses {
				ip := net.ParseIP(addr)
				if ip == nil {
					continue
				}
				if ip.To4() != nil {
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    uint32(ttl),
						},
						A: ip,
					}
					m.Answer = append(m.Answer, rr)
				}
			}
			return
		}
	}
}

func (s *Server) handleTXTQuery(q dns.Question, m *dns.Msg, domain string) {
	name := strings.TrimSuffix(q.Name, ".")
	name = strings.TrimSuffix(name, domain)

	records := s.provider.GetAllRecords()
	for _, rec := range records {
		if rec.Hostname == name {
			ttl := s.getTTL(rec.Status)

			var txt []string
			txt = append(txt, fmt.Sprintf("status=%s", rec.Status))
			txt = append(txt, fmt.Sprintf("last_seen=%d", rec.LastSeen.Unix()))

			if rec.IsStatic {
				txt = append(txt, "static=true")
			}

			services := make([]string, 0, len(rec.Services))
			for name, svc := range rec.Services {
				services = append(services, fmt.Sprintf("%s:%d", name, svc.Port))
			}
			if len(services) > 0 {
				txt = append(txt, "services="+strings.Join(services, ","))
			}

			rr := &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    uint32(ttl),
				},
				Txt: txt,
			}
			m.Answer = append(m.Answer, rr)
			return
		}
	}
}

func (s *Server) handleSRVQuery(q dns.Question, m *dns.Msg, domain string) {
	// SRV format: _service._proto.domain
	// e.g., _www._tcp.disco.
	parts := strings.Split(strings.TrimSuffix(q.Name, "."), ".")
	if len(parts) < 4 {
		return
	}

	serviceName := strings.TrimPrefix(parts[0], "_")
	proto := strings.TrimPrefix(parts[1], "_")
	queryDomain := strings.Join(parts[2:], ".") + "."

	if queryDomain != domain {
		return
	}

	records := s.provider.GetAllRecords()
	for _, rec := range records {
		for svcName, svc := range rec.Services {
			if svcName == serviceName && svc.Protocol == proto {
				ttl := s.getTTL(rec.Status)
				rr := &dns.SRV{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeSRV,
						Class:  dns.ClassINET,
						Ttl:    uint32(ttl),
					},
					Priority: 10,
					Weight:   60,
					Port:     uint16(svc.Port),
					Target:   rec.Hostname + "." + domain,
				}
				m.Answer = append(m.Answer, rr)
			}
		}
	}
}

func (s *Server) handleCNAMEQuery(q dns.Question, m *dns.Msg, domain string) {
	// CNAME for service aliases: www.host.disco -> host.disco
	name := strings.TrimSuffix(q.Name, ".")
	name = strings.TrimSuffix(name, domain)

	records := s.provider.GetAllRecords()
	for _, rec := range records {
		for svcName := range rec.Services {
			alias := svcName + "." + rec.Hostname
			if name == alias {
				ttl := s.getTTL(rec.Status)
				rr := &dns.CNAME{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeCNAME,
						Class:  dns.ClassINET,
						Ttl:    uint32(ttl),
					},
					Target: rec.Hostname + "." + domain,
				}
				m.Answer = append(m.Answer, rr)
				return
			}
		}
	}
}

func (s *Server) handlePTRQuery(q dns.Question, m *dns.Msg) {
	// Reverse lookup: 10.1.168.192.in-addr.arpa -> host.disco
	parts := strings.Split(strings.TrimSuffix(q.Name, "."), ".")
	if len(parts) < 4 {
		return
	}

	// Reconstruct IP from reverse notation
	ipStr := fmt.Sprintf("%s.%s.%s.%s", parts[3], parts[2], parts[1], parts[0])

	records := s.provider.GetAllRecords()
	for _, rec := range records {
		for _, addr := range rec.Addresses {
			if addr == ipStr {
				ttl := s.getTTL(rec.Status)
				rr := &dns.PTR{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypePTR,
						Class:  dns.ClassINET,
						Ttl:    uint32(ttl),
					},
					Ptr: rec.Hostname + "." + s.config.Domain + ".",
				}
				m.Answer = append(m.Answer, rr)
				return
			}
		}
	}
}

func (s *Server) getTTL(status string) int {
	if status == "stale" {
		return s.config.TTLStale
	}
	return s.config.TTLHealthy
}
