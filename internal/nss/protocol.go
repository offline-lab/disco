package nss

import (
	"encoding/json"
	"fmt"
)

// MessageType represents the type of NSS query
type MessageType string

const (
	DefaultSocketPath = "/run/disco.sock"

	QueryByName       MessageType = "QUERY_BY_NAME"
	QueryByAddr       MessageType = "QUERY_BY_ADDR"
	QueryList         MessageType = "LIST"
	QueryListHosts    MessageType = "LIST_HOSTS"
	QueryListServices MessageType = "LIST_SERVICES"

	HostsList     MessageType = "HOSTS_LIST"
	HostsShow     MessageType = "HOSTS_SHOW"
	HostsForget   MessageType = "HOSTS_FORGET"
	HostsMarkLost MessageType = "HOSTS_MARK_LOST"

	ServicesList   MessageType = "SERVICES_LIST"
	ServicesShow   MessageType = "SERVICES_SHOW"
	ServicesForget MessageType = "SERVICES_FORGET"

	ResponseOK       MessageType = "OK"
	ResponseNotFound MessageType = "NOTFOUND"
	ResponseError    MessageType = "ERROR"
)

// Query represents an NSS query from the libnss module
type Query struct {
	Type      MessageType `json:"type"`
	Name      string      `json:"name,omitempty"`
	Addr      string      `json:"addr,omitempty"`
	Family    int         `json:"family,omitempty"`
	RequestID string      `json:"request_id"`
}

type Response struct {
	Type      MessageType     `json:"type"`
	RequestID string          `json:"request_id"`
	Name      string          `json:"name,omitempty"`
	Aliases   []string        `json:"aliases,omitempty"`
	AddrType  int             `json:"addr_type,omitempty"`
	AddrLen   int             `json:"addr_len,omitempty"`
	Addrs     []string        `json:"addrs,omitempty"`
	Error     string          `json:"error,omitempty"`
	Records   []byte          `json:"records,omitempty"`
	Count     int             `json:"count,omitempty"`
	Hosts     []HostHealth    `json:"hosts,omitempty"`
	Services  []ServiceHealth `json:"services,omitempty"`
}

type HostHealth struct {
	Hostname    string            `json:"hostname"`
	Addresses   []string          `json:"addresses"`
	Status      string            `json:"status"`
	Services    map[string]string `json:"services"`
	LastSeen    int64             `json:"last_seen"`
	LastSeenAgo string            `json:"last_seen_ago"`
	IsStatic    bool              `json:"is_static"`
}

type ServiceHealth struct {
	Name     string   `json:"name"`
	Protocol string   `json:"protocol"`
	Port     int      `json:"port"`
	Hosts    []string `json:"hosts"`
	Status   string   `json:"status"`
}

// Record represents a host record stored in the daemon
type Record struct {
	Hostname  string
	Aliases   []string
	Addresses []string
	Timestamp int64
	FirstSeen int64
	TTL       int64
	Services  map[string]string
	Status    HostStatus
	IsStatic  bool
}

type HostStatus string

const (
	StatusHealthy HostStatus = "healthy"
	StatusStale   HostStatus = "stale"
	StatusLost    HostStatus = "lost"
	StatusStatic  HostStatus = "static"
)

// MarshalQuery converts a Query to JSON
func MarshalQuery(q *Query) ([]byte, error) {
	return json.Marshal(q)
}

// UnmarshalQuery parses JSON into a Query
func UnmarshalQuery(data []byte) (*Query, error) {
	var q Query
	err := json.Unmarshal(data, &q)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// MarshalResponse converts a Response to JSON
func MarshalResponse(r *Response) ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalResponse parses JSON into a Response
func UnmarshalResponse(data []byte) (*Response, error) {
	var r Response
	err := json.Unmarshal(data, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// NewOKResponse creates a successful response
func NewOKResponse(requestID, name string, addrs []string) *Response {
	return &Response{
		Type:      ResponseOK,
		RequestID: requestID,
		Name:      name,
		Addrs:     addrs,
	}
}

// NewNotFoundResponse creates a not found response
func NewNotFoundResponse(requestID string) *Response {
	return &Response{
		Type:      ResponseNotFound,
		RequestID: requestID,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(requestID, errMsg string) *Response {
	return &Response{
		Type:      ResponseError,
		RequestID: requestID,
		Error:     errMsg,
	}
}

func (r *Record) String() string {
	return fmt.Sprintf("%s -> %v", r.Hostname, r.Addresses)
}
