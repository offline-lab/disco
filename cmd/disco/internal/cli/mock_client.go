package cli

import (
	"time"

	"github.com/offline-lab/disco/internal/nss"
)

type MockDaemonClient struct {
	Response  *nss.Response
	Error     error
	CallCount int
}

func NewMockDaemonClient(response *nss.Response, err error) *MockDaemonClient {
	return &MockDaemonClient{
		Response:  response,
		Error:     err,
		CallCount: 0,
	}
}

func (m *MockDaemonClient) Query(query *nss.Query) (*nss.Response, error) {
	m.CallCount++
	if m.Error != nil {
		return nil, m.Error
	}

	if m.Response == nil {
		m.Response = &nss.Response{}
	}

	return m.Response, m.Error
}

func (m *MockDaemonClient) WithTimeout(timeout time.Duration) *MockDaemonClient {
	return m
}
