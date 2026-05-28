package dns

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

type mockProvider struct {
	records []DNSRecord
}

func (m *mockProvider) GetAllRecords() []DNSRecord {
	return m.records
}

func testServer(records []DNSRecord) *Server {
	return NewServer(&Config{
		Enabled:    true,
		Port:       15354,
		Domain:     "disco",
		TTLHealthy: 60,
		TTLStale:   10,
	}, &mockProvider{records: records})
}

func testRecords() []DNSRecord {
	return []DNSRecord{
		{
			Hostname:  "web1",
			Addresses: []string{"192.168.1.10", "10.0.0.1"},
			Services: map[string]ServiceInfo{
				"www":  {Port: 80, Protocol: "tcp"},
				"smtp": {Port: 25, Protocol: "tcp"},
			},
			Status:   "healthy",
			LastSeen: time.Now(),
		},
		{
			Hostname:  "mail1",
			Addresses: []string{"192.168.1.20"},
			Services: map[string]ServiceInfo{
				"smtp": {Port: 25, Protocol: "tcp"},
			},
			Status:   "stale",
			LastSeen: time.Now().Add(-10 * time.Minute),
			IsStatic: true,
		},
	}
}

func queryServer(s *Server, name string, qtype uint16) *dns.Msg {
	r := new(dns.Msg)
	r.SetQuestion(name, qtype)

	w := &testResponseWriter{}
	s.handleQuery(w, r)
	return w.msg
}

func reverseQueryServer(s *Server, name string) *dns.Msg {
	r := new(dns.Msg)
	r.SetQuestion(name, dns.TypePTR)

	w := &testResponseWriter{}
	s.handleReverse(w, r)
	return w.msg
}

type testResponseWriter struct {
	msg *dns.Msg
}

func (w *testResponseWriter) LocalAddr() net.Addr       { return nil }
func (w *testResponseWriter) RemoteAddr() net.Addr      { return nil }
func (w *testResponseWriter) WriteMsg(m *dns.Msg) error { w.msg = m; return nil }
func (w *testResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *testResponseWriter) Close() error              { return nil }
func (w *testResponseWriter) TsigStatus() error         { return nil }
func (w *testResponseWriter) TsigTimersOnly(bool)       {}
func (w *testResponseWriter) Hijack()                    {}

func TestHandleAQuery(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "web1.disco.", dns.TypeA)

	if len(resp.Answer) != 2 {
		t.Fatalf("expected 2 A records, got %d", len(resp.Answer))
	}

	ips := map[string]bool{}
	for _, rr := range resp.Answer {
		a, ok := rr.(*dns.A)
		if !ok {
			t.Fatalf("expected A record, got %T", rr)
		}
		ips[a.A.String()] = true
		if a.Hdr.Ttl != 60 {
			t.Errorf("expected TTL 60 (healthy), got %d", a.Hdr.Ttl)
		}
	}
	if !ips["192.168.1.10"] || !ips["10.0.0.1"] {
		t.Errorf("expected IPs 192.168.1.10 and 10.0.0.1, got %v", ips)
	}
}

func TestHandleAQuery_StaleHost(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "mail1.disco.", dns.TypeA)

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 A record, got %d", len(resp.Answer))
	}
	a := resp.Answer[0].(*dns.A)
	if a.Hdr.Ttl != 10 {
		t.Errorf("expected TTL 10 (stale), got %d", a.Hdr.Ttl)
	}
}

func TestHandleAQuery_NotFound(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "unknown.disco.", dns.TypeA)

	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 answers for unknown host, got %d", len(resp.Answer))
	}
}

func TestHandleAQuery_WrongDomain(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "web1.other.", dns.TypeA)

	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 answers for wrong domain, got %d", len(resp.Answer))
	}
}

func TestHandleTXTQuery(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "web1.disco.", dns.TypeTXT)

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 TXT record, got %d", len(resp.Answer))
	}

	txt, ok := resp.Answer[0].(*dns.TXT)
	if !ok {
		t.Fatalf("expected TXT record, got %T", resp.Answer[0])
	}

	found := map[string]bool{}
	for _, s := range txt.Txt {
		found[s] = true
	}
	if !found["status=healthy"] {
		t.Errorf("expected status=healthy in TXT, got %v", txt.Txt)
	}
}

func TestHandleTXTQuery_StaticHost(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "mail1.disco.", dns.TypeTXT)

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 TXT record, got %d", len(resp.Answer))
	}

	txt := resp.Answer[0].(*dns.TXT)
	found := map[string]bool{}
	for _, s := range txt.Txt {
		found[s] = true
	}
	if !found["static=true"] {
		t.Errorf("expected static=true in TXT, got %v", txt.Txt)
	}
	if !found["status=stale"] {
		t.Errorf("expected status=stale in TXT, got %v", txt.Txt)
	}
}

func TestHandleSRVQuery(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "_www._tcp.disco.", dns.TypeSRV)

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 SRV record, got %d", len(resp.Answer))
	}

	srv, ok := resp.Answer[0].(*dns.SRV)
	if !ok {
		t.Fatalf("expected SRV record, got %T", resp.Answer[0])
	}
	if srv.Port != 80 {
		t.Errorf("expected port 80, got %d", srv.Port)
	}
	if srv.Target != "web1.disco." {
		t.Errorf("expected target web1.disco., got %s", srv.Target)
	}
}

func TestHandleSRVQuery_MultipleHosts(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "_smtp._tcp.disco.", dns.TypeSRV)

	if len(resp.Answer) != 2 {
		t.Fatalf("expected 2 SRV records (web1+mail1 both have smtp), got %d", len(resp.Answer))
	}

	targets := map[string]bool{}
	for _, rr := range resp.Answer {
		srv := rr.(*dns.SRV)
		targets[srv.Target] = true
		if srv.Port != 25 {
			t.Errorf("expected port 25, got %d", srv.Port)
		}
	}
	if !targets["web1.disco."] || !targets["mail1.disco."] {
		t.Errorf("expected targets web1.disco. and mail1.disco., got %v", targets)
	}
}

func TestHandleSRVQuery_NoMatch(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "_ssh._tcp.disco.", dns.TypeSRV)

	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 SRV records for unknown service, got %d", len(resp.Answer))
	}
}

func TestHandleCNAMEQuery(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "www.web1.disco.", dns.TypeCNAME)

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 CNAME record, got %d", len(resp.Answer))
	}

	cname, ok := resp.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record, got %T", resp.Answer[0])
	}
	if cname.Target != "web1.disco." {
		t.Errorf("expected target web1.disco., got %s", cname.Target)
	}
}

func TestHandleCNAMEQuery_NoMatch(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "ssh.web1.disco.", dns.TypeCNAME)

	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 CNAME records for non-service alias, got %d", len(resp.Answer))
	}
}

func TestHandlePTRQuery(t *testing.T) {
	s := testServer(testRecords())

	resp := reverseQueryServer(s, "10.1.168.192.in-addr.arpa.")

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 PTR record, got %d", len(resp.Answer))
	}

	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		t.Fatalf("expected PTR record, got %T", resp.Answer[0])
	}
	if ptr.Ptr != "web1.disco." {
		t.Errorf("expected web1.disco., got %s", ptr.Ptr)
	}
}

func TestHandlePTRQuery_NotFound(t *testing.T) {
	s := testServer(testRecords())

	resp := reverseQueryServer(s, "99.99.99.99.in-addr.arpa.")

	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 PTR records for unknown IP, got %d", len(resp.Answer))
	}
}

func TestHandlePTRQuery_SecondAddress(t *testing.T) {
	s := testServer(testRecords())

	resp := reverseQueryServer(s, "1.0.0.10.in-addr.arpa.")

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 PTR record, got %d", len(resp.Answer))
	}
	ptr := resp.Answer[0].(*dns.PTR)
	if ptr.Ptr != "web1.disco." {
		t.Errorf("expected web1.disco., got %s", ptr.Ptr)
	}
}

func TestGetTTL(t *testing.T) {
	s := testServer(nil)

	if ttl := s.getTTL("healthy"); ttl != 60 {
		t.Errorf("expected 60 for healthy, got %d", ttl)
	}
	if ttl := s.getTTL("stale"); ttl != 10 {
		t.Errorf("expected 10 for stale, got %d", ttl)
	}
	if ttl := s.getTTL("lost"); ttl != 60 {
		t.Errorf("expected 60 for unknown status, got %d", ttl)
	}
}

func TestEmptyRecords(t *testing.T) {
	s := testServer(nil)

	resp := queryServer(s, "web1.disco.", dns.TypeA)
	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 answers with nil provider records, got %d", len(resp.Answer))
	}
}

func TestAuthoritative(t *testing.T) {
	s := testServer(testRecords())

	resp := queryServer(s, "web1.disco.", dns.TypeA)
	if !resp.Authoritative {
		t.Error("expected authoritative response")
	}
}
