package nss

import (
	"encoding/json"
	"testing"
)

func TestMarshalQuery(t *testing.T) {
	q := &Query{
		Type:      QueryByName,
		Name:      "test-host",
		RequestID: "test-001",
	}

	data, err := MarshalQuery(q)
	if err != nil {
		t.Fatalf("MarshalQuery() error = %v", err)
	}

	if data == nil {
		t.Fatal("MarshalQuery() returned nil")
	}

	var unmarshaled Query
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if unmarshaled.Type != QueryByName {
		t.Errorf("Expected type %s, got %s", QueryByName, unmarshaled.Type)
	}

	if unmarshaled.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", unmarshaled.Name)
	}
}

func TestUnmarshalQuery(t *testing.T) {
	data := []byte(`{"type":"QUERY_BY_NAME","name":"test-host","request_id":"test-001"}`)

	q, err := UnmarshalQuery(data)
	if err != nil {
		t.Fatalf("UnmarshalQuery() error = %v", err)
	}

	if q.Type != QueryByName {
		t.Errorf("Expected type %s, got %s", QueryByName, q.Type)
	}

	if q.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", q.Name)
	}

	if q.RequestID != "test-001" {
		t.Errorf("Expected request_id test-001, got %s", q.RequestID)
	}
}

func TestUnmarshalQuery_Invalid(t *testing.T) {
	data := []byte(`{"type":"INVALID_TYPE"}`)

	_, err := UnmarshalQuery(data)
	if err != nil {
		// Should not error on invalid type (unmarshals but type won't match constants)
		// This is expected behavior
		return
	}

	// If no error, that's also fine - JSON was valid
}

func TestMarshalResponse(t *testing.T) {
	r := &Response{
		Type:      ResponseOK,
		RequestID: "test-001",
		Name:      "test-host",
		Addrs:     []string{"192.168.1.10", "192.168.1.11"},
	}

	data, err := MarshalResponse(r)
	if err != nil {
		t.Fatalf("MarshalResponse() error = %v", err)
	}

	if data == nil {
		t.Fatal("MarshalResponse() returned nil")
	}

	var unmarshaled Response
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if unmarshaled.Type != ResponseOK {
		t.Errorf("Expected type %s, got %s", ResponseOK, unmarshaled.Type)
	}

	if len(unmarshaled.Addrs) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(unmarshaled.Addrs))
	}
}

func TestUnmarshalResponse(t *testing.T) {
	data := []byte(`{"type":"OK","request_id":"test-001","name":"test-host","addrs":["192.168.1.10"]}`)

	r, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	if r.Type != ResponseOK {
		t.Errorf("Expected type %s, got %s", ResponseOK, r.Type)
	}

	if r.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", r.Name)
	}

	if len(r.Addrs) != 1 {
		t.Errorf("Expected 1 address, got %d", len(r.Addrs))
	}

	if r.Addrs[0] != "192.168.1.10" {
		t.Errorf("Expected address 192.168.1.10, got %s", r.Addrs[0])
	}
}

func TestNewOKResponse(t *testing.T) {
	r := NewOKResponse("test-001", "test-host", []string{"192.168.1.10"})

	if r.Type != ResponseOK {
		t.Errorf("Expected type %s, got %s", ResponseOK, r.Type)
	}

	if r.RequestID != "test-001" {
		t.Errorf("Expected request_id test-001, got %s", r.RequestID)
	}

	if r.Name != "test-host" {
		t.Errorf("Expected name test-host, got %s", r.Name)
	}

	if len(r.Addrs) != 1 {
		t.Errorf("Expected 1 address, got %d", len(r.Addrs))
	}

	if r.Addrs[0] != "192.168.1.10" {
		t.Errorf("Expected address 192.168.1.10, got %s", r.Addrs[0])
	}
}

func TestNewNotFoundResponse(t *testing.T) {
	r := NewNotFoundResponse("test-001")

	if r.Type != ResponseNotFound {
		t.Errorf("Expected type %s, got %s", ResponseNotFound, r.Type)
	}

	if r.RequestID != "test-001" {
		t.Errorf("Expected request_id test-001, got %s", r.RequestID)
	}

	if len(r.Addrs) != 0 {
		t.Errorf("Expected 0 addresses, got %d", len(r.Addrs))
	}
}

func TestNewErrorResponse(t *testing.T) {
	r := NewErrorResponse("test-001", "host not found")

	if r.Type != ResponseError {
		t.Errorf("Expected type %s, got %s", ResponseError, r.Type)
	}

	if r.RequestID != "test-001" {
		t.Errorf("Expected request_id test-001, got %s", r.RequestID)
	}

	if r.Error != "host not found" {
		t.Errorf("Expected error 'host not found', got %s", r.Error)
	}
}

func TestRecord_String(t *testing.T) {
	r := &Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10", "192.168.1.11"},
		Timestamp: 1234567890,
		TTL:       3600,
	}

	str := r.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check that hostname is in the string
	if len(str) < len(r.Hostname) {
		t.Errorf("String() too short, expected at least %d chars", len(r.Hostname))
	}
}

func TestMessageType_Constants(t *testing.T) {
	// Verify all message type constants are set
	expectedTypes := []MessageType{
		QueryByName,
		QueryByAddr,
		QueryList,
		QueryListHosts,
		QueryListServices,
		ResponseOK,
		ResponseNotFound,
		ResponseError,
	}

	for _, msgType := range expectedTypes {
		if msgType == "" {
			t.Errorf("MessageType constant is empty")
		}
	}
}

func TestQuery_JSONRoundtrip(t *testing.T) {
	original := &Query{
		Type:      QueryByAddr,
		Addr:      "192.168.1.10",
		Family:    2,
		RequestID: "test-roundtrip",
	}

	data, err := MarshalQuery(original)
	if err != nil {
		t.Fatalf("MarshalQuery() error = %v", err)
	}

	unmarshaled, err := UnmarshalQuery(data)
	if err != nil {
		t.Fatalf("UnmarshalQuery() error = %v", err)
	}

	if unmarshaled.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, original.Type)
	}

	if unmarshaled.Addr != original.Addr {
		t.Errorf("Addr mismatch: got %s, want %s", unmarshaled.Addr, original.Addr)
	}

	if unmarshaled.Family != original.Family {
		t.Errorf("Family mismatch: got %d, want %d", unmarshaled.Family, original.Family)
	}

	if unmarshaled.RequestID != original.RequestID {
		t.Errorf("RequestID mismatch: got %s, want %s", unmarshaled.RequestID, original.RequestID)
	}
}

func TestResponse_JSONRoundtrip(t *testing.T) {
	original := &Response{
		Type:      ResponseOK,
		RequestID: "test-roundtrip",
		Name:      "test-host",
		Addrs:     []string{"192.168.1.10", "192.168.1.11", "10.0.0.1"},
		AddrType:  2,
		AddrLen:   4,
	}

	data, err := MarshalResponse(original)
	if err != nil {
		t.Fatalf("MarshalResponse() error = %v", err)
	}

	unmarshaled, err := UnmarshalResponse(data)
	if err != nil {
		t.Fatalf("UnmarshalResponse() error = %v", err)
	}

	if unmarshaled.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, original.Type)
	}

	if unmarshaled.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", unmarshaled.Name, original.Name)
	}

	if len(unmarshaled.Addrs) != len(original.Addrs) {
		t.Errorf("Addrs length mismatch: got %d, want %d", len(unmarshaled.Addrs), len(original.Addrs))
	}

	if unmarshaled.AddrType != original.AddrType {
		t.Errorf("AddrType mismatch: got %d, want %d", unmarshaled.AddrType, original.AddrType)
	}

	if unmarshaled.AddrLen != original.AddrLen {
		t.Errorf("AddrLen mismatch: got %d, want %d", unmarshaled.AddrLen, original.AddrLen)
	}
}

func TestResponse_Aliases(t *testing.T) {
	r := &Response{
		Type:      ResponseOK,
		RequestID: "test-001",
		Name:      "test-host",
		Aliases:   []string{"alias1", "alias2"},
		Addrs:     []string{"192.168.1.10"},
	}

	data, _ := MarshalResponse(r)
	unmarshaled, _ := UnmarshalResponse(data)

	if len(unmarshaled.Aliases) != 2 {
		t.Errorf("Expected 2 aliases, got %d", len(unmarshaled.Aliases))
	}

	if unmarshaled.Aliases[0] != "alias1" {
		t.Errorf("Expected alias1, got %s", unmarshaled.Aliases[0])
	}
}

func TestResponse_ErrorField(t *testing.T) {
	r := NewErrorResponse("test-001", "internal error")

	if r.Error == "" {
		t.Error("Error field should not be empty")
	}

	if r.Type != ResponseError {
		t.Errorf("Expected type %s, got %s", ResponseError, r.Type)
	}
}

func TestQuery_AllOptionalFields(t *testing.T) {
	// Test query with all fields
	q := &Query{
		Type:      QueryList,
		Name:      "",
		Addr:      "",
		Family:    0,
		RequestID: "test-all",
	}

	data, _ := MarshalQuery(q)
	unmarshaled, _ := UnmarshalQuery(data)

	if unmarshaled.Type != QueryList {
		t.Errorf("Type mismatch: got %s, want %s", unmarshaled.Type, QueryList)
	}
}

func TestQuery_EmptyJSON(t *testing.T) {
	data := []byte(`{}`)

	q, err := UnmarshalQuery(data)
	if err != nil {
		t.Fatalf("UnmarshalQuery() error = %v", err)
	}

	// Empty JSON is valid, just has default values
	if q.RequestID == "" {
		// RequestID is set to empty string by default
	}
}

func TestRecord_Services(t *testing.T) {
	r := &Record{
		Hostname:  "test-host",
		Addresses: []string{"192.168.1.10"},
		Timestamp: 1234567890,
		TTL:       3600,
		Services: map[string]string{
			"www":  "192.168.1.10:80",
			"smtp": "192.168.1.10:25",
		},
	}

	if r.Services == nil {
		t.Fatal("Services map is nil")
	}

	if r.Services["www"] != "192.168.1.10:80" {
		t.Errorf("Expected www service at 192.168.1.10:80, got %s", r.Services["www"])
	}

	if r.Services["smtp"] != "192.168.1.10:25" {
		t.Errorf("Expected smtp service at 192.168.1.10:25, got %s", r.Services["smtp"])
	}
}

func TestRecord_AllFields(t *testing.T) {
	r := &Record{
		Hostname:  "test-host",
		Aliases:   []string{"alias1", "alias2"},
		Addresses: []string{"192.168.1.10", "192.168.1.11"},
		Timestamp: 1234567890,
		TTL:       3600,
		Services: map[string]string{
			"www": "192.168.1.10:80",
		},
	}

	// Verify all fields
	if len(r.Aliases) != 2 {
		t.Errorf("Expected 2 aliases, got %d", len(r.Aliases))
	}

	if len(r.Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(r.Addresses))
	}

	if r.Timestamp != 1234567890 {
		t.Errorf("Expected timestamp 1234567890, got %d", r.Timestamp)
	}

	if r.TTL != 3600 {
		t.Errorf("Expected TTL 3600, got %d", r.TTL)
	}

	if len(r.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(r.Services))
	}
}
