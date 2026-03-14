package service

import (
	"testing"
	"time"
)

func TestNewDetector_NilConfig(t *testing.T) {
	detector := NewDetector(nil, 60*time.Second)
	if detector == nil {
		t.Fatal("NewDetector() returned nil for nil config")
	}
}

func TestDetector_GetServices_Empty(t *testing.T) {
	detector := NewDetector(map[string][]int{}, 60*time.Second)
	services := detector.GetServices()

	if services == nil {
		t.Fatal("GetServices() returned nil")
	}
	if len(services) != 0 {
		t.Errorf("GetServices() count = %d, want 0", len(services))
	}
}

func TestDetector_GetServiceCount(t *testing.T) {
	portMapping := map[string][]int{
		"www": {80, 443},
		"ssh": {22},
	}

	detector := NewDetector(portMapping, 60*time.Second)

	if count := detector.GetServiceCount(); count != 0 {
		t.Errorf("GetServiceCount() = %d, want 0 (no services yet)", count)
	}
}

func TestDetector_AddService(t *testing.T) {
	detector := NewDetector(map[string][]int{}, 60*time.Second)

	detector.mu.Lock()
	detector.services["test:80"] = ServiceInfo{
		Name: "test",
		Port: 80,
		Addr: "192.168.1.1",
	}
	detector.mu.Unlock()

	services := detector.GetServices()
	if len(services) != 1 {
		t.Errorf("GetServices() count = %d, want 1", len(services))
	}
	if services[0].Name != "test" {
		t.Errorf("Service name = %s, want test", services[0].Name)
	}
}

func TestDetector_MultipleServices(t *testing.T) {
	detector := NewDetector(map[string][]int{}, 60*time.Second)

	detector.mu.Lock()
	detector.services["www:80"] = ServiceInfo{Name: "www", Port: 80, Addr: "1.1.1.1"}
	detector.services["ssh:22"] = ServiceInfo{Name: "ssh", Port: 22, Addr: "1.1.1.1"}
	detector.services["smtp:25"] = ServiceInfo{Name: "smtp", Port: 25, Addr: "1.1.1.1"}
	detector.mu.Unlock()

	if count := detector.GetServiceCount(); count != 3 {
		t.Errorf("GetServiceCount() = %d, want 3", count)
	}
}

func TestDetector_Stop_ClearsServices(t *testing.T) {
	detector := NewDetector(map[string][]int{}, 60*time.Second)

	detector.mu.Lock()
	detector.services["test:80"] = ServiceInfo{Name: "test", Port: 80}
	detector.mu.Unlock()

	detector.Stop()

	if count := detector.GetServiceCount(); count != 0 {
		t.Errorf("GetServiceCount() after Stop() = %d, want 0", count)
	}
}

func TestDetector_PortMapping(t *testing.T) {
	portMapping := map[string][]int{
		"www": {80, 443, 8080},
		"ssh": {22},
		"dns": {53},
	}

	detector := NewDetector(portMapping, 60*time.Second)

	if len(detector.portMapping) != 3 {
		t.Errorf("portMapping count = %d, want 3", len(detector.portMapping))
	}
}

func TestDetector_Interval(t *testing.T) {
	interval := 30 * time.Second
	detector := NewDetector(nil, interval)

	if detector.interval != interval {
		t.Errorf("interval = %v, want %v", detector.interval, interval)
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
	if svc.Addr != "192.168.1.1" {
		t.Errorf("Addr = %s, want 192.168.1.1", svc.Addr)
	}
}
