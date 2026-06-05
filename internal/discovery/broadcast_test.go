package discovery

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBroadcastMessage_Marshal(t *testing.T) {
	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: "test-123",
		Timestamp: time.Now().Unix(),
		Hostname:  "testhost",
		IPs:       []string{"192.168.1.1"},
		Services: []ServiceInfo{
			{Name: "www", Port: 80, Addr: "192.168.1.1"},
		},
		TTL: 3600,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded BroadcastMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Hostname != "testhost" {
		t.Errorf("Hostname = %s, want testhost", decoded.Hostname)
	}
	if decoded.Type != MessageAnnounce {
		t.Errorf("Type = %s, want ANNOUNCE", decoded.Type)
	}
	if decoded.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", decoded.TTL)
	}
}

func TestTimeAnnounceMessage_Marshal(t *testing.T) {
	msg := &TimeAnnounceMessage{
		Type:      MessageTimeAnnounce,
		MessageID: "time-123",
		Timestamp: time.Now().UnixNano(),
		ClockInfo: ClockInfo{
			Stratum:        1,
			Precision:      -20,
			RootDelay:      0.0,
			RootDispersion: 0.0001,
			ReferenceID:    "GPS",
			ReferenceTime:  time.Now().UnixNano(),
		},
		LeapIndicator: 0,
		SourceID:      "gps-1",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded TimeAnnounceMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SourceID != "gps-1" {
		t.Errorf("SourceID = %s, want gps-1", decoded.SourceID)
	}
	if decoded.ClockInfo.Stratum != 1 {
		t.Errorf("Stratum = %d, want 1", decoded.ClockInfo.Stratum)
	}
}

func TestServiceInfo_Fields(t *testing.T) {
	svc := ServiceInfo{
		Name: "www",
		Port: 80,
		Addr: "192.168.1.1",
	}

	if svc.Name != "www" {
		t.Errorf("Name = %s, want www", svc.Name)
	}
	if svc.Port != 80 {
		t.Errorf("Port = %d, want 80", svc.Port)
	}
}

func TestServiceInfo_Aliases_MarshalRoundtrip(t *testing.T) {
	svc := ServiceInfo{
		Name:    "http",
		Port:    80,
		Addr:    "192.168.1.1",
		Aliases: []string{"shop.local", "blog.local"},
	}

	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded ServiceInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(decoded.Aliases) != 2 {
		t.Fatalf("Expected 2 aliases, got %d", len(decoded.Aliases))
	}
	if decoded.Aliases[0] != "shop.local" || decoded.Aliases[1] != "blog.local" {
		t.Errorf("Aliases = %v, want [shop.local blog.local]", decoded.Aliases)
	}
}

func TestServiceInfo_Aliases_OmitEmpty(t *testing.T) {
	svc := ServiceInfo{Name: "ssh", Port: 22, Addr: "192.168.1.1"}

	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(data) != `{"name":"ssh","port":22,"addr":"192.168.1.1"}` {
		// aliases field must be absent when nil
		var m map[string]interface{}
		_ = json.Unmarshal(data, &m)
		if _, ok := m["aliases"]; ok {
			t.Error("aliases field present in JSON when it should be omitted")
		}
	}
}

func TestBroadcastMessage_MachineID_Roundtrip(t *testing.T) {
	fullID := "abcd1234abcd1234abcd1234abcd1234"
	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: "test-mid",
		Timestamp: 1000,
		Hostname:  "node1",
		MachineID: fullID,
		IPs:       []string{"192.168.1.1"},
		TTL:       3600,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded BroadcastMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.MachineID != fullID {
		t.Errorf("MachineID = %q, want %q", decoded.MachineID, fullID)
	}
}

func TestBroadcastMessage_MachineID_OmitEmpty(t *testing.T) {
	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: "test-no-mid",
		Timestamp: 1000,
		Hostname:  "oldnode",
		IPs:       []string{"192.168.1.2"},
		TTL:       3600,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if _, ok := m["machine_id"]; ok {
		t.Error("machine_id field present in JSON when it should be omitted")
	}
}

func TestBroadcastMessage_ServiceAliases_Roundtrip(t *testing.T) {
	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: "test-aliases",
		Timestamp: 1000,
		Hostname:  "web.local",
		IPs:       []string{"192.168.1.1"},
		Services: []ServiceInfo{
			{Name: "http", Port: 80, Addr: "192.168.1.1", Aliases: []string{"shop.local", "api.local"}},
		},
		TTL: 3600,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded BroadcastMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(decoded.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(decoded.Services))
	}
	if len(decoded.Services[0].Aliases) != 2 {
		t.Fatalf("Expected 2 aliases, got %d", len(decoded.Services[0].Aliases))
	}
	if decoded.Services[0].Aliases[0] != "shop.local" {
		t.Errorf("First alias = %q, want shop.local", decoded.Services[0].Aliases[0])
	}
}

func TestClockInfo_Fields(t *testing.T) {
	ci := ClockInfo{
		Stratum:        1,
		Precision:      -20,
		RootDelay:      0.001,
		RootDispersion: 0.0001,
		ReferenceID:    "GPS",
		ReferenceTime:  1234567890,
	}

	if ci.Stratum != 1 {
		t.Errorf("Stratum = %d, want 1", ci.Stratum)
	}
	if ci.ReferenceID != "GPS" {
		t.Errorf("ReferenceID = %s, want GPS", ci.ReferenceID)
	}
}

func TestMessageType_Constants(t *testing.T) {
	if MessageAnnounce != "ANNOUNCE" {
		t.Errorf("MessageAnnounce = %s, want ANNOUNCE", MessageAnnounce)
	}
	if MessageQuery != "QUERY" {
		t.Errorf("MessageQuery = %s, want QUERY", MessageQuery)
	}
	if MessageResponse != "RESPONSE" {
		t.Errorf("MessageResponse = %s, want RESPONSE", MessageResponse)
	}
	if MessageTimeAnnounce != "TIME_ANNOUNCE" {
		t.Errorf("MessageTimeAnnounce = %s, want TIME_ANNOUNCE", MessageTimeAnnounce)
	}
}

func TestRateLimiter_Defaults(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	if rl.rate != defaultRate {
		t.Errorf("rate = %d, want %d", rl.rate, defaultRate)
	}
	if rl.maxBurst != defaultMaxBurst {
		t.Errorf("maxBurst = %d, want %d", rl.maxBurst, defaultMaxBurst)
	}
}

func TestRateLimiter_NegativeValues(t *testing.T) {
	rl := NewRateLimiter(-1, -5)
	if rl.rate != defaultRate {
		t.Errorf("rate = %d, want %d", rl.rate, defaultRate)
	}
	if rl.maxBurst != defaultMaxBurst {
		t.Errorf("maxBurst = %d, want %d", rl.maxBurst, defaultMaxBurst)
	}
}

func TestDuplicateFilter_Stop(t *testing.T) {
	df := NewDuplicateFilter(5 * time.Minute)
	df.Stop()
}
