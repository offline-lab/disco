package service

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/logging"
)

type Detector struct {
	portMapping map[string][]int
	interval    time.Duration
	services    map[string]ServiceInfo
	mu          sync.RWMutex
}

type ServiceInfo struct {
	Name string
	Port int
	Addr string
}

func NewDetector(portMapping map[string][]int, interval time.Duration) *Detector {
	return &Detector{
		portMapping: portMapping,
		interval:    interval,
		services:    make(map[string]ServiceInfo),
	}
}

func (d *Detector) Start(stopChan chan struct{}) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	d.scan()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			d.scan()
		}
	}
}

func (d *Detector) scan() {
	d.mu.Lock()
	defer d.mu.Unlock()

	oldCount := len(d.services)
	newServices := make(map[string]ServiceInfo)

	for serviceName, ports := range d.portMapping {
		for _, port := range ports {
			if addr, ok := d.checkPort(port); ok {
				key := fmt.Sprintf("%s:%d", serviceName, port)
				newServices[key] = ServiceInfo{
					Name: serviceName,
					Port: port,
					Addr: addr,
				}
			}
		}
	}

	d.services = newServices

	if len(newServices) != oldCount {
		logging.Debug("Service scan completed", map[string]interface{}{
			"services_found": len(newServices),
			"previous_count": oldCount,
		})
	}
}

func (d *Detector) checkPort(port int) (string, bool) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	if err != nil {
		return "", false
	}
	defer func() { _ = conn.Close() }()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	ip := localAddr.IP.To4()
	if ip == nil {
		return "", true
	}

	return ip.String(), true
}

func (d *Detector) GetServices() []ServiceInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	services := make([]ServiceInfo, 0, len(d.services))
	for _, svc := range d.services {
		services = append(services, svc)
	}

	return services
}

func (d *Detector) GetServiceCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.services)
}

func (d *Detector) Stop() {
	d.mu.Lock()
	d.services = make(map[string]ServiceInfo)
	d.mu.Unlock()
}
