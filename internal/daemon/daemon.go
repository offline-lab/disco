package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	dnsserver "github.com/offline-lab/disco/internal/dns"
	"github.com/offline-lab/disco/internal/logging"
	"github.com/offline-lab/disco/internal/notify"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/offline-lab/disco/internal/security"
	"github.com/offline-lab/disco/internal/service"
	"github.com/offline-lab/disco/internal/timesync"
)

type Daemon struct {
	config          *config.Config
	hostname        string
	store           *RecordStore
	socket          *SocketServer
	announcer       *discovery.Announcer
	listener        *discovery.Listener
	detector        *service.Detector
	timeSync        *timesync.TimeSyncService
	dnsServer       *dnsserver.Server
	stopChan        chan struct{}
	dynamicServices []config.ServiceConfig
	dynamicMu       sync.Mutex
}

// New creates a new daemon instance
func New(cfg *config.Config) (*Daemon, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	store := NewRecordStore(cfg.Daemon.RecordTTL, &cfg.Health, cfg.StaticHosts)
	socket := NewSocketServer(cfg.Daemon.SocketPath, store)

	var keyManager *security.KeyManager
	if cfg.Security.Enabled {
		keyManager, err = security.NewKeyManager(cfg.Security.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize key manager: %w", err)
		}
		if cfg.Security.TrustedPeers != "" {
			if err := loadTrustedPeers(keyManager, cfg.Security.TrustedPeers); err != nil {
				return nil, fmt.Errorf("failed to load trusted peers: %w", err)
			}
		}
	}

	d := &Daemon{
		config:   cfg,
		hostname: hostname,
		store:    store,
		socket:   socket,
		stopChan: make(chan struct{}),
	}

	if cfg.Discovery.Enabled {
		announcer, err := discovery.NewAnnouncer(
			cfg.Network.BroadcastAddr,
			hostname,
			cfg.Daemon.BroadcastInterval,
			keyManager,
			cfg.Network.Interfaces,
		)
		if err != nil {
			return nil, err
		}
		d.announcer = announcer

		listener, err := discovery.NewListener(cfg.Network.BroadcastAddr, keyManager, cfg.Security.RequireSigned)
		if err != nil {
			return nil, err
		}
		d.listener = listener

		if cfg.Discovery.DetectServices {
			detector := service.NewDetector(
				cfg.Discovery.ServicePortMapping,
				cfg.Discovery.ScanInterval,
			)
			d.detector = detector
		}
	}

	if cfg.TimeSync.Enabled {
		d.timeSync = timesync.NewTimeSyncService(&cfg.TimeSync, keyManager)
		d.socket.SetTimeSync(d.timeSync)
	}

	if cfg.DNS.Enabled {
		dnsConfig := &dnsserver.Config{
			Enabled:       cfg.DNS.Enabled,
			Port:          cfg.DNS.Port,
			Domain:        cfg.DNS.Domain,
			BindAddresses: cfg.DNS.BindAddresses,
			TTLHealthy:    cfg.DNS.TTLHealthy,
			TTLStale:      cfg.DNS.TTLStale,
		}
		d.dnsServer = dnsserver.NewServer(dnsConfig, d.store)
	}

	return d, nil
}

func loadTrustedPeers(km *security.KeyManager, path string) error {
	km.AddTrustedPeerByID(km.GetPublicKey(), km.GetPrivateKey())

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Warn("Trusted peers file not found, using self only", map[string]interface{}{"path": path})
			return nil
		}
		return fmt.Errorf("failed to read trusted peers file: %w", err)
	}

	var peers []struct {
		Hostname   string `json:"hostname"`
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
	}

	if err := json.Unmarshal(data, &peers); err != nil {
		return fmt.Errorf("failed to parse trusted peers file: %w", err)
	}

	for _, peer := range peers {
		if peer.Hostname != "" && peer.PublicKey != "" && peer.PrivateKey != "" {
			km.AddTrustedPeerByID(peer.PublicKey, peer.PrivateKey)
			logging.Info("Added trusted peer", map[string]interface{}{"hostname": peer.Hostname})
		}
	}

	return nil
}

// Run starts daemon
func (d *Daemon) Run() error {
	logging.Info("Starting disco-daemon", nil)

	if d.announcer != nil {
		for _, svc := range d.config.Services {
			ips := d.announcer.GetLocalIPs()
			addr := ""
			if len(ips) > 0 {
				addr = ips[0]
			}
			d.announcer.AddService(svc.Name, svc.Port, addr, svc.Aliases)
		}

		go d.announcer.Start(d.stopChan)
		logging.Info("Announcer started", nil)
	}

	if d.listener != nil {
		go d.listener.Start(d.stopChan)
		go d.processDiscoveryMessages()
		logging.Info("Listener started", nil)
	}

	if d.detector != nil {
		go d.detector.Start(d.stopChan)
		go d.updateServiceAnnouncements(d.stopChan)
		logging.Info("Service detector started", nil)
	}

	if d.timeSync != nil {
		go d.timeSync.Start()
		go d.processTimeMessages()
		logging.Info("Time sync service started", nil)
	}

	d.socket.SetAnnounceService(d.AnnounceService)

	if err := d.socket.Start(); err != nil {
		return fmt.Errorf("failed to start socket server: %w", err)
	}
	logging.Info("Socket server started", map[string]interface{}{"socket_path": d.config.Daemon.SocketPath})

	if d.dnsServer != nil {
		if err := d.dnsServer.Start(); err != nil {
			logging.Error("Failed to start DNS server", err, nil)
		} else {
			logging.Info("DNS server started", map[string]interface{}{
				"port":   d.config.DNS.Port,
				"domain": d.config.DNS.Domain,
				"bind":   d.config.DNS.BindAddresses,
			})
		}
	}

	d.selfRegister()
	notify.Ready()
	logging.Info("Daemon ready", nil)

	d.waitForShutdown()

	return nil
}

// processDiscoveryMessages handles incoming discovery messages
func (d *Daemon) processDiscoveryMessages() {
	for msg := range d.listener.Messages() {
		d.handleDiscoveryMessage(msg)
	}
}

// processTimeMessages handles incoming time messages
func (d *Daemon) processTimeMessages() {
	for msg := range d.listener.TimeMessages() {
		d.timeSync.ProcessMessage(msg)
	}
}

// handleDiscoveryMessage processes a discovery message
func (d *Daemon) handleDiscoveryMessage(msg *discovery.BroadcastMessage) {
	record := &nss.Record{
		Hostname:  msg.Hostname,
		Addresses: msg.IPs,
		Timestamp: msg.Timestamp,
		TTL:       msg.TTL,
		Services:  make(map[string]string),
	}

	seen := make(map[string]bool)
	for _, svc := range msg.Services {
		record.Services[svc.Name] = fmt.Sprintf("%s:%d", svc.Addr, svc.Port)
		for _, alias := range svc.Aliases {
			if !seen[alias] {
				record.Aliases = append(record.Aliases, alias)
				seen[alias] = true
			}
		}
	}

	d.store.AddOrUpdate(record)
}

// updateServiceAnnouncements updates the announcer with detected services
func (d *Daemon) updateServiceAnnouncements(stopChan chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			logging.Info("Service announcement updater stopped", nil)
			return
		case <-ticker.C:
			services := d.detector.GetServices()
			for _, svc := range services {
				d.announcer.AddService(svc.Name, svc.Port, svc.Addr, nil)
			}
		}
	}
}

// selfRegister creates/updates the daemon's own record in the store so local aliases resolve immediately.
func (d *Daemon) selfRegister() {
	var ips []string
	if d.announcer != nil {
		ips = d.announcer.GetLocalIPs()
	}

	addr := ""
	if len(ips) > 0 {
		addr = ips[0]
	}

	record := &nss.Record{
		Hostname:  d.hostname,
		Addresses: ips,
		Timestamp: time.Now().Unix(),
		TTL:       int64(d.config.Daemon.RecordTTL.Seconds()),
		Services:  make(map[string]string),
	}

	d.dynamicMu.Lock()
	allServices := make([]config.ServiceConfig, len(d.config.Services), len(d.config.Services)+len(d.dynamicServices))
	copy(allServices, d.config.Services)
	allServices = append(allServices, d.dynamicServices...)
	d.dynamicMu.Unlock()

	seen := make(map[string]bool)
	for _, svc := range allServices {
		record.Services[svc.Name] = fmt.Sprintf("%s:%d", addr, svc.Port)
		for _, alias := range svc.Aliases {
			if !seen[alias] {
				record.Aliases = append(record.Aliases, alias)
				seen[alias] = true
			}
		}
	}

	d.store.AddOrUpdate(record)
}

// AnnounceService adds a service to the daemon's advertisements and triggers an immediate broadcast.
func (d *Daemon) AnnounceService(name string, port int, aliases []string) error {
	d.dynamicMu.Lock()
	d.dynamicServices = append(d.dynamicServices, config.ServiceConfig{
		Name:    name,
		Port:    port,
		Aliases: aliases,
	})
	d.dynamicMu.Unlock()

	if d.announcer != nil {
		ips := d.announcer.GetLocalIPs()
		addr := ""
		if len(ips) > 0 {
			addr = ips[0]
		}
		d.announcer.AddService(name, port, addr, aliases)
		d.announcer.BroadcastNow()
	}

	d.selfRegister()
	logging.Info("Service announced", map[string]interface{}{"name": name, "port": port, "aliases": aliases})
	return nil
}

// waitForShutdown waits for shutdown signal
func (d *Daemon) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	notify.Stopping()
	logging.Info("Received signal, shutting down", map[string]interface{}{"signal": sig.String()})

	close(d.stopChan)

	if d.announcer != nil {
		d.announcer.Stop()
	}
	if d.listener != nil {
		d.listener.Stop()
	}
	if d.detector != nil {
		d.detector.Stop()
	}
	if d.timeSync != nil {
		d.timeSync.Stop()
	}
	if d.socket != nil {
		d.socket.Stop()
	}
	if d.dnsServer != nil {
		d.dnsServer.Stop()
	}
	if d.store != nil {
		d.store.Stop()
	}

	_ = os.Remove(d.config.Daemon.SocketPath)

	logging.Info("Shutdown complete", nil)
}
