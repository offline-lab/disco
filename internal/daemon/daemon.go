package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/nss"
	"github.com/offline-lab/disco/internal/security"
	"github.com/offline-lab/disco/internal/service"
)

// Daemon is not main daemon struct
type Daemon struct {
	config    *config.Config
	store     *RecordStore
	socket    *SocketServer
	announcer *discovery.Announcer
	listener  *discovery.Listener
	detector  *service.Detector
	stopChan  chan struct{}
}

// New creates a new daemon instance
func New(cfg *config.Config) (*Daemon, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	store := NewRecordStore(cfg.Daemon.RecordTTL)
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

	return d, nil
}

func loadTrustedPeers(km *security.KeyManager, path string) error {
	km.AddTrustedPeerByID(km.GetPublicKey(), km.GetPrivateKey())

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Trusted peers file not found: %s (using self only)", path)
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
			log.Printf("Added trusted peer: %s", peer.Hostname)
		}
	}

	return nil
}

// Run starts daemon
func (d *Daemon) Run() error {
	log.Printf("Starting nss-daemon...")

	if d.announcer != nil {
		go d.announcer.Start(d.stopChan)
		log.Printf("Announcer started")
	}

	if d.listener != nil {
		go d.listener.Start(d.stopChan)
		go d.processDiscoveryMessages()
		log.Printf("Listener started")
	}

	if d.detector != nil {
		go d.detector.Start(d.stopChan)
		go d.updateServiceAnnouncements(d.stopChan)
		log.Printf("Service detector started")
	}

	go d.socket.Start()
	log.Printf("Socket server started on %s", d.config.Daemon.SocketPath)

	d.waitForShutdown()

	return nil
}

// processDiscoveryMessages handles incoming discovery messages
func (d *Daemon) processDiscoveryMessages() {
	for msg := range d.listener.Messages() {
		d.handleDiscoveryMessage(msg)
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

	for _, svc := range msg.Services {
		record.Services[svc.Name] = fmt.Sprintf("%s:%d", svc.Addr, svc.Port)
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
			log.Printf("Service announcement updater stopped")
			return
		case <-ticker.C:
			services := d.detector.GetServices()
			for _, svc := range services {
				d.announcer.AddService(svc.Name, svc.Port, svc.Addr)
			}
		}
	}
}

// waitForShutdown waits for shutdown signal
func (d *Daemon) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)

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
	if d.socket != nil {
		d.socket.Stop()
	}
	if d.store != nil {
		d.store.Stop()
	}

	os.Remove(d.config.Daemon.SocketPath)

	log.Printf("Shutdown complete")
}
