package service

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Detector scans for services running on the local host
type Detector struct {
	portMapping map[string][]int
	interval    time.Duration
	services    map[string]ServiceInfo
	mu          sync.RWMutex
}

// ServiceInfo represents a detected service
type ServiceInfo struct {
	Name string
	Port int
	Addr string
}

// NewDetector creates a new service detector
func NewDetector(portMapping map[string][]int, interval time.Duration) *Detector {
	return &Detector{
		portMapping: portMapping,
		interval:    interval,
		services:    make(map[string]ServiceInfo),
	}
}

// Start begins scanning for services
func (d *Detector) Start(stopChan chan struct{}) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			d.scan()
		}
	}
}

// scan scans for services
func (d *Detector) scan() {
	d.mu.Lock()
	defer d.mu.Unlock()

	newServices := make(map[string]ServiceInfo)

	for serviceName, ports := range d.portMapping {
		for _, port := range ports {
			if d.isPortOpen(port) {
				addr := d.getLocalAddr(port)
				if addr != "" {
					key := fmt.Sprintf("%s:%d", serviceName, port)
					newServices[key] = ServiceInfo{
						Name: serviceName,
						Port: port,
						Addr: addr,
					}
				}
			}
		}
	}

	d.services = newServices
}

// isPortOpen checks if a port is open
func (d *Detector) isPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getLocalAddr returns the local IP address for a service
func (d *Detector) getLocalAddr(port int) string {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	ip := localAddr.IP.To4()
	if ip == nil {
		return ""
	}

	return ip.String()
}

// GetServices returns all detected services
func (d *Detector) GetServices() []ServiceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	services := make([]ServiceInfo, 0, len(d.services))
	for _, svc := range d.services {
		services = append(services, svc)
	}

	return services
}

// Stop stops the detector
func (d *Detector) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.services = make(map[string]ServiceInfo)
}
