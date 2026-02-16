package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/security"
)

// MessageType represents the type of broadcast message
type MessageType string

const (
	MessageAnnounce MessageType = "ANNOUNCE"
	MessageQuery    MessageType = "QUERY"
	MessageResponse MessageType = "RESPONSE"
)

// BroadcastMessage represents a broadcast announcement
type BroadcastMessage struct {
	Type      MessageType               `json:"type"`
	MessageID string                    `json:"message_id"`
	Timestamp int64                     `json:"timestamp"`
	Hostname  string                    `json:"hostname"`
	IPs       []string                  `json:"ips"`
	Services  []ServiceInfo             `json:"services"`
	Signature *security.MessageSecurity `json:"signature,omitempty"`
	TTL       int64                     `json:"ttl"`
}

// ServiceInfo represents a service running on a node
type ServiceInfo struct {
	Name string `json:"name"`
	Port int    `json:"port"`
	Addr string `json:"addr"`
}

// Announcer handles sending broadcast announcements
type Announcer struct {
	broadcastAddr string
	hostname      string
	interval      time.Duration
	conn          *net.UDPConn
	services      map[string]ServiceInfo
	rateLimiter   *RateLimiter
	keyManager    *security.KeyManager
}

// Listener handles receiving broadcast messages
type Listener struct {
	broadcastAddr   string
	messageChan     chan *BroadcastMessage
	conns           []*net.UDPConn
	duplicateFilter *DuplicateFilter
	keyManager      *security.KeyManager
	requireSigned   bool
}

// NewAnnouncer creates a new broadcast announcer
func NewAnnouncer(broadcastAddr, hostname string, interval time.Duration, keyManager *security.KeyManager) (*Announcer, error) {
	_, err := net.ResolveUDPAddr("udp4", broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	return &Announcer{
		broadcastAddr: broadcastAddr,
		hostname:      hostname,
		interval:      interval,
		conn:          conn,
		services:      make(map[string]ServiceInfo),
		rateLimiter:   NewRateLimiter(10, 10),
		keyManager:    keyManager,
	}, nil
}

// Start begins broadcasting announcements
func (a *Announcer) Start(stopChan chan struct{}) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			a.broadcast()
		}
	}
}

// broadcast sends an announcement message
func (a *Announcer) broadcast() {
	if !a.rateLimiter.Allow() {
		return
	}

	msg := a.createAnnouncement()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	broadcastAddr, err := net.ResolveUDPAddr("udp4", a.broadcastAddr)
	if err != nil {
		return
	}

	a.conn.WriteToUDP(data, broadcastAddr)

	_, port, err := net.SplitHostPort(a.broadcastAddr)
	if err != nil {
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				if len(ipnet.Mask) == 4 {
					broadcast := make(net.IP, 4)
					for i := 0; i < 4; i++ {
						broadcast[i] = ipnet.IP.To4()[i] | ^ipnet.Mask[i]
					}
					targetAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", broadcast.String(), port))
					if err == nil {
						a.conn.WriteToUDP(data, targetAddr)
					}
				}
			}
		}
	}
}

// createAnnouncement creates an announcement message
func (a *Announcer) createAnnouncement() *BroadcastMessage {
	services := make([]ServiceInfo, 0, len(a.services))
	for _, svc := range a.services {
		services = append(services, svc)
	}

	ips := a.getLocalIPs()

	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: fmt.Sprintf("%s-%d", a.hostname, time.Now().UnixNano()),
		Timestamp: time.Now().Unix(),
		Hostname:  a.hostname,
		IPs:       ips,
		Services:  services,
		TTL:       3600,
	}

	if a.keyManager != nil {
		msgWithoutSig := &BroadcastMessage{
			Type:      msg.Type,
			MessageID: msg.MessageID,
			Timestamp: msg.Timestamp,
			Hostname:  msg.Hostname,
			IPs:       msg.IPs,
			Services:  msg.Services,
			TTL:       msg.TTL,
		}
		sigData, err := json.Marshal(msgWithoutSig)
		if err == nil {
			sig, err := a.keyManager.Sign(sigData)
			if err == nil {
				msg.Signature = sig
			}
		}
	}

	return msg
}

// getLocalIPs returns local IP addresses
func (a *Announcer) getLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}

// AddService adds a service to be announced
func (a *Announcer) AddService(name string, port int, addr string) {
	a.services[name] = ServiceInfo{
		Name: name,
		Port: port,
		Addr: addr,
	}
}

// RemoveService removes a service from announcements
func (a *Announcer) RemoveService(name string) {
	delete(a.services, name)
}

// Stop stops the announcer
func (a *Announcer) Stop() {
	a.conn.Close()
}

// NewListener creates a new broadcast listener
func NewListener(broadcastAddr string, keyManager *security.KeyManager, requireSigned bool) (*Listener, error) {
	_, port, err := net.SplitHostPort(broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse broadcast address: %w", err)
	}

	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	conns := []*net.UDPConn{conn}

	return &Listener{
		broadcastAddr:   broadcastAddr,
		messageChan:     make(chan *BroadcastMessage, 100),
		conns:           conns,
		duplicateFilter: NewDuplicateFilter(5 * time.Minute),
		keyManager:      keyManager,
		requireSigned:   requireSigned,
	}, nil
}

// Start begins listening for broadcast messages
func (l *Listener) Start(stopChan chan struct{}) {
	var wg sync.WaitGroup

	for _, conn := range l.conns {
		wg.Add(1)
		go func(c *net.UDPConn) {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				select {
				case <-stopChan:
					return
				default:
					c.SetReadDeadline(time.Now().Add(1 * time.Second))
					n, _, err := c.ReadFromUDP(buf)
					if err != nil {
						continue
					}

					var msg BroadcastMessage
					if err := json.Unmarshal(buf[:n], &msg); err != nil {
						continue
					}

					if l.duplicateFilter.Seen(msg.MessageID) {
						continue
					}

					if l.requireSigned && msg.Signature == nil {
						continue
					}

					if l.keyManager != nil && msg.Signature != nil {
						msgWithoutSig := &BroadcastMessage{
							Type:      msg.Type,
							MessageID: msg.MessageID,
							Timestamp: msg.Signature.Timestamp,
							Hostname:  msg.Hostname,
							IPs:       msg.IPs,
							Services:  msg.Services,
							TTL:       msg.TTL,
						}
						sigData, _ := json.Marshal(msgWithoutSig)
						if !l.keyManager.Verify(sigData, msg.Signature) {
							continue
						}
					}

					select {
					case l.messageChan <- &msg:
					case <-stopChan:
						return
					}
				}
			}
		}(conn)
	}

	<-stopChan
	wg.Wait()
}

// Messages returns the channel for received messages
func (l *Listener) Messages() <-chan *BroadcastMessage {
	return l.messageChan
}

// Stop stops the listener
func (l *Listener) Stop() {
	if l.duplicateFilter != nil {
		l.duplicateFilter.Stop()
	}
	close(l.messageChan)
	for _, conn := range l.conns {
		conn.Close()
	}
}
