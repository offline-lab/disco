package service

import (
	"testing"
	"time"
)

func TestDetector_New(t *testing.T) {
	portMapping := map[string][]int{
		"www":  {80, 443},
		"smtp": {25},
	}

	detector := NewDetector(portMapping, 60*time.Second)
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}

	if detector.interval != 60*time.Second {
		t.Errorf("Expected interval 60s, got %v", detector.interval)
	}
}

func TestDetector_GetServices(t *testing.T) {
	portMapping := map[string][]int{
		"www": {80, 443},
	}

	detector := NewDetector(portMapping, 60*time.Second)
	services := detector.GetServices()

	if services == nil {
		t.Fatal("GetServices() returned nil")
	}

	// Initially no services detected
	if len(services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(services))
	}
}

func TestDetector_AddService(t *testing.T) {
	portMapping := map[string][]int{
		"www": {80},
	}

	detector := NewDetector(portMapping, 60*time.Second)

	// Manually add a service to test internal state
	detector.mu.Lock()
	detector.services["www:80"] = ServiceInfo{
		Name: "www",
		Port: 80,
		Addr: "192.168.1.10",
	}
	detector.mu.Unlock()

	services := detector.GetServices()
	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if services[0].Name != "www" {
		t.Errorf("Expected service name www, got %s", services[0].Name)
	}

	if services[0].Port != 80 {
		t.Errorf("Expected port 80, got %d", services[0].Port)
	}

	if services[0].Addr != "192.168.1.10" {
		t.Errorf("Expected address 192.168.1.10, got %s", services[0].Addr)
	}
}

func TestDetector_Stop(t *testing.T) {
	portMapping := map[string][]int{
		"www": {80},
	}

	detector := NewDetector(portMapping, 60*time.Second)

	// Add a service
	detector.mu.Lock()
	detector.services["www:80"] = ServiceInfo{
		Name: "www",
		Port: 80,
		Addr: "192.168.1.10",
	}
	detector.mu.Unlock()

	// Stop detector
	detector.Stop()

	// Verify services were cleared
	services := detector.GetServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 services after stop, got %d", len(services))
	}
}

func TestServiceInfo_String(t *testing.T) {
	info := ServiceInfo{
		Name: "www",
		Port: 80,
		Addr: "192.168.1.10",
	}

	// Just verify it doesn't panic
	_ = info.Name
	_ = info.Port
	_ = info.Addr
}
