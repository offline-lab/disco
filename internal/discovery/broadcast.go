package discovery

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/offline-lab/disco/internal/logging"
	"github.com/offline-lab/disco/internal/security"
)

type MessageType string

const (
	MessageAnnounce     MessageType = "ANNOUNCE"
	MessageQuery        MessageType = "QUERY"
	MessageResponse     MessageType = "RESPONSE"
	MessageTimeAnnounce MessageType = "TIME_ANNOUNCE"
)

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

type ServiceInfo struct {
	Name string `json:"name"`
	Port int    `json:"port"`
	Addr string `json:"addr"`
}

type ClockInfo struct {
	Stratum        int     `json:"stratum"`
	Precision      int     `json:"precision"`
	RootDelay      float64 `json:"root_delay"`
	RootDispersion float64 `json:"root_dispersion"`
	ReferenceID    string  `json:"reference_id"`
	ReferenceTime  int64   `json:"reference_time"`
}

type TimeAnnounceMessage struct {
	Type          MessageType               `json:"type"`
	MessageID     string                    `json:"message_id"`
	Timestamp     int64                     `json:"timestamp"`
	ClockInfo     ClockInfo                 `json:"clock_info"`
	LeapIndicator int                       `json:"leap_indicator"`
	SourceID      string                    `json:"source_id"`
	Signature     *security.MessageSecurity `json:"signature,omitempty"`
}

type Announcer struct {
	broadcastAddr string
	hostname      string
	interval      time.Duration
	conn          *net.UDPConn
	services      map[string]ServiceInfo
	servicesMu    sync.RWMutex
	rateLimiter   *RateLimiter
	keyManager    *security.KeyManager

	cachedBroadcastAddr *net.UDPAddr
	cachedPort          string
	ifaceFilter         map[string]bool
	ifaceCache          []net.Interface
	ifaceCacheTime      time.Time
	ifaceCacheTTL       time.Duration

	bufPool sync.Pool
}

type Listener struct {
	broadcastAddr   string
	messageChan     chan *BroadcastMessage
	timeMessageChan chan *TimeAnnounceMessage
	conns           []*net.UDPConn
	duplicateFilter *DuplicateFilter
	keyManager      *security.KeyManager
	requireSigned   bool

	bufPool sync.Pool
}

func NewAnnouncer(broadcastAddr, hostname string, interval time.Duration, keyManager *security.KeyManager, interfaces []string) (*Announcer, error) {
	addr, err := net.ResolveUDPAddr("udp4", broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	_, port, err := net.SplitHostPort(broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse broadcast address: %w", err)
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	ifaceFilter := make(map[string]bool, len(interfaces))
	for _, name := range interfaces {
		ifaceFilter[name] = true
	}

	return &Announcer{
		broadcastAddr:       broadcastAddr,
		hostname:            hostname,
		interval:            interval,
		conn:                conn,
		services:            make(map[string]ServiceInfo),
		rateLimiter:         NewRateLimiter(10, 10),
		keyManager:          keyManager,
		cachedBroadcastAddr: addr,
		cachedPort:          port,
		ifaceFilter:         ifaceFilter,
		ifaceCacheTTL:       30 * time.Second,
		bufPool: sync.Pool{
			New: func() interface{} { return make([]byte, 0, 2048) },
		},
	}, nil
}

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

func (a *Announcer) broadcast() {
	if !a.rateLimiter.Allow() {
		return
	}

	msg := a.createAnnouncement()

	data, err := json.Marshal(msg)
	if err != nil {
		logging.Error("Failed to marshal broadcast message", err, nil)
		return
	}

	a.conn.WriteToUDP(data, a.cachedBroadcastAddr)

	ifaces := a.getCachedInterfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if len(a.ifaceFilter) > 0 && !a.ifaceFilter[iface.Name] {
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
					ip4 := ipnet.IP.To4()
					for i := 0; i < 4; i++ {
						broadcast[i] = ip4[i] | ^ipnet.Mask[i]
					}
					targetAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", broadcast.String(), a.cachedPort))
					if err == nil {
						a.conn.WriteToUDP(data, targetAddr)
					}
				}
			}
		}
	}
}

func (a *Announcer) getCachedInterfaces() []net.Interface {
	now := time.Now()
	if a.ifaceCache != nil && now.Sub(a.ifaceCacheTime) < a.ifaceCacheTTL {
		return a.ifaceCache
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return a.ifaceCache
	}

	a.ifaceCache = ifaces
	a.ifaceCacheTime = now
	return ifaces
}

func (a *Announcer) createAnnouncement() *BroadcastMessage {
	a.servicesMu.RLock()
	services := make([]ServiceInfo, 0, len(a.services))
	for _, svc := range a.services {
		services = append(services, svc)
	}
	a.servicesMu.RUnlock()

	ips := a.getLocalIPs()
	now := time.Now()
	nowUnix := now.Unix()
	nowNano := now.UnixNano()

	msg := &BroadcastMessage{
		Type:      MessageAnnounce,
		MessageID: fmt.Sprintf("%s-%d", a.hostname, nowNano),
		Timestamp: nowUnix,
		Hostname:  a.hostname,
		IPs:       ips,
		Services:  services,
		TTL:       3600,
	}

	if a.keyManager != nil {
		sigData, err := json.Marshal(struct {
			Type      MessageType   `json:"type"`
			MessageID string        `json:"message_id"`
			Timestamp int64         `json:"timestamp"`
			Hostname  string        `json:"hostname"`
			IPs       []string      `json:"ips"`
			Services  []ServiceInfo `json:"services"`
			TTL       int64         `json:"ttl"`
		}{
			Type:      msg.Type,
			MessageID: msg.MessageID,
			Timestamp: msg.Timestamp,
			Hostname:  msg.Hostname,
			IPs:       msg.IPs,
			Services:  msg.Services,
			TTL:       msg.TTL,
		})
		if err != nil {
			logging.Error("Failed to marshal message for signing", err, nil)
			return msg
		}
		sig, err := a.keyManager.Sign(sigData)
		if err != nil {
			logging.Error("Failed to sign message", err, nil)
			return msg
		}
		msg.Signature = sig
	}

	return msg
}

func (a *Announcer) getLocalIPs() []string {
	var ips []string

	ifaces := a.getCachedInterfaces()
	for _, iface := range ifaces {
		if len(a.ifaceFilter) > 0 && !a.ifaceFilter[iface.Name] {
			continue
		}

		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

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

func (a *Announcer) AddService(name string, port int, addr string) {
	a.servicesMu.Lock()
	a.services[name] = ServiceInfo{
		Name: name,
		Port: port,
		Addr: addr,
	}
	a.servicesMu.Unlock()
}

func (a *Announcer) RemoveService(name string) {
	a.servicesMu.Lock()
	delete(a.services, name)
	a.servicesMu.Unlock()
}

func (a *Announcer) Stop() {
	a.conn.Close()
}

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

	return &Listener{
		broadcastAddr:   broadcastAddr,
		messageChan:     make(chan *BroadcastMessage, 100),
		timeMessageChan: make(chan *TimeAnnounceMessage, 100),
		conns:           []*net.UDPConn{conn},
		duplicateFilter: NewDuplicateFilter(5 * time.Minute),
		keyManager:      keyManager,
		requireSigned:   requireSigned,
		bufPool: sync.Pool{
			New: func() interface{} { return make([]byte, 4096) },
		},
	}, nil
}

type rawMessage struct {
	Type      string `json:"type"`
	MessageID string `json:"message_id"`
}

func (l *Listener) Start(stopChan chan struct{}) {
	var wg sync.WaitGroup

	for _, conn := range l.conns {
		wg.Add(1)
		go func(c *net.UDPConn) {
			defer wg.Done()
			buf := l.bufPool.Get().([]byte)
			defer l.bufPool.Put(buf)

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

					data := buf[:n]

					var raw rawMessage
					if err := json.Unmarshal(data, &raw); err != nil {
						continue
					}

					if l.duplicateFilter.Seen(raw.MessageID) {
						continue
					}

					if raw.Type == string(MessageTimeAnnounce) {
						l.handleTimeMessage(data, stopChan)
						continue
					}

					var msg BroadcastMessage
					if err := json.Unmarshal(data, &msg); err != nil {
						continue
					}

					if l.requireSigned && msg.Signature == nil {
						continue
					}

					if l.keyManager != nil && msg.Signature != nil {
						sigData, _ := json.Marshal(struct {
							Type      MessageType   `json:"type"`
							MessageID string        `json:"message_id"`
							Timestamp int64         `json:"timestamp"`
							Hostname  string        `json:"hostname"`
							IPs       []string      `json:"ips"`
							Services  []ServiceInfo `json:"services"`
							TTL       int64         `json:"ttl"`
						}{
							Type:      msg.Type,
							MessageID: msg.MessageID,
							Timestamp: msg.Timestamp,
							Hostname:  msg.Hostname,
							IPs:       msg.IPs,
							Services:  msg.Services,
							TTL:       msg.TTL,
						})
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

func (l *Listener) handleTimeMessage(data []byte, stopChan chan struct{}) {
	var timeMsg TimeAnnounceMessage
	if err := json.Unmarshal(data, &timeMsg); err != nil {
		return
	}

	if l.requireSigned && timeMsg.Signature == nil && l.keyManager != nil {
		return
	}

	if l.keyManager != nil && timeMsg.Signature != nil {
		sigData, _ := json.Marshal(struct {
			Type          MessageType `json:"type"`
			MessageID     string      `json:"message_id"`
			Timestamp     int64       `json:"timestamp"`
			ClockInfo     ClockInfo   `json:"clock_info"`
			LeapIndicator int         `json:"leap_indicator"`
			SourceID      string      `json:"source_id"`
		}{
			Type:          timeMsg.Type,
			MessageID:     timeMsg.MessageID,
			Timestamp:     timeMsg.Timestamp,
			ClockInfo:     timeMsg.ClockInfo,
			LeapIndicator: timeMsg.LeapIndicator,
			SourceID:      timeMsg.SourceID,
		})
		if !l.keyManager.Verify(sigData, timeMsg.Signature) {
			return
		}
	}

	select {
	case l.timeMessageChan <- &timeMsg:
	case <-stopChan:
	}
}

func (l *Listener) Messages() <-chan *BroadcastMessage {
	return l.messageChan
}

func (l *Listener) TimeMessages() <-chan *TimeAnnounceMessage {
	return l.timeMessageChan
}

func (l *Listener) Stop() {
	if l.duplicateFilter != nil {
		l.duplicateFilter.Stop()
	}
	close(l.messageChan)
	close(l.timeMessageChan)
	for _, conn := range l.conns {
		conn.Close()
	}
}
