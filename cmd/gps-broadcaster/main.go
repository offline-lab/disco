// disco-gps-broadcaster - GPS time broadcaster for disco-daemon
// Build: make disco-gps-broadcaster
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	device     = flag.String("device", "/dev/ttyACM0", "GPS serial device")
	baudRate   = flag.Int("baud", 9600, "Baud rate (ignored on Unix)")
	broadcast  = flag.String("broadcast", "255.255.255.255:5354", "Broadcast address")
	sourceID   = flag.String("id", "", "Source ID (default: hostname)")
	interfaces = flag.String("interfaces", "", "Comma-separated list of interfaces")
	verbose    = flag.Bool("v", false, "Verbose output")
)

type TimeAnnounceMessage struct {
	Type          string    `json:"type"`
	MessageID     string    `json:"message_id"`
	Timestamp     int64     `json:"timestamp"`
	ClockInfo     ClockInfo `json:"clock_info"`
	LeapIndicator int       `json:"leap_indicator"`
	SourceID      string    `json:"source_id"`
}

type ClockInfo struct {
	Stratum        int     `json:"stratum"`
	Precision      int     `json:"precision"`
	RootDelay      float64 `json:"root_delay"`
	RootDispersion float64 `json:"root_dispersion"`
	ReferenceID    string  `json:"reference_id"`
	ReferenceTime  int64   `json:"reference_time"`
}

type BackoffManager struct {
	mu           sync.Mutex
	attempts     int
	lastHasFix   bool
	lastSats     int
	lastInterval time.Duration
}

func NewBackoffManager() *BackoffManager {
	return &BackoffManager{}
}

func (b *BackoffManager) NextInterval(hasFix bool, sats int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	signalImproved := false

	if hasFix && !b.lastHasFix {
		signalImproved = true
	}
	if sats > b.lastSats+2 && sats > 0 {
		signalImproved = true
	}

	b.lastHasFix = hasFix
	b.lastSats = sats

	if hasFix {
		b.attempts = 0
		b.lastInterval = 16 * time.Second
		return b.lastInterval
	}

	if signalImproved {
		b.attempts = 0
	}

	b.attempts++

	switch {
	case b.attempts <= 3:
		b.lastInterval = 10 * time.Second
	case b.attempts <= 6:
		b.lastInterval = 30 * time.Second
	case b.attempts <= 10:
		b.lastInterval = 2 * time.Minute
	case b.attempts <= 15:
		b.lastInterval = 5 * time.Minute
	case b.attempts <= 20:
		b.lastInterval = 10 * time.Minute
	case b.attempts <= 25:
		b.lastInterval = 20 * time.Minute
	case b.attempts <= 30:
		b.lastInterval = 30 * time.Minute
	default:
		b.lastInterval = time.Hour
	}

	return b.lastInterval
}

func (b *BackoffManager) CurrentInterval() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lastInterval
}

type GPSReader struct {
	device     string
	baudRate   int
	serial     *os.File
	reader     *bufio.Reader
	mu         sync.Mutex
	hasFix     bool
	gpsTime    time.Time
	satellites int
}

func NewGPSReader(device string, baudRate int) *GPSReader {
	return &GPSReader{
		device:   device,
		baudRate: baudRate,
	}
}

func (g *GPSReader) Open() error {
	f, err := os.OpenFile(g.device, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", g.device, err)
	}
	g.serial = f
	g.reader = bufio.NewReader(f)
	return nil
}

func (g *GPSReader) Close() {
	if g.serial != nil {
		g.serial.Close()
	}
}

func (g *GPSReader) SetReadDeadline(d time.Time) error {
	return g.serial.SetReadDeadline(d)
}

func (g *GPSReader) ReadLine() (string, error) {
	line, err := g.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (g *GPSReader) ParseNMEA(line string) {
	if len(line) < 6 || !strings.HasPrefix(line, "$") {
		return
	}

	fields := strings.Split(line, ",")
	if len(fields) < 1 {
		return
	}

	sentence := fields[0]

	switch sentence {
	case "$GPRMC", "$GNRMC":
		g.parseRMC(fields)
	case "$GPGGA", "$GNGGA":
		g.parseGGA(fields)
	}
}

func (g *GPSReader) parseRMC(fields []string) {
	if len(fields) < 10 {
		return
	}

	status := fields[2]
	if status != "A" {
		g.mu.Lock()
		g.hasFix = false
		g.mu.Unlock()
		return
	}

	dateStr := fields[9]
	timeStr := fields[1]

	if len(dateStr) < 6 || len(timeStr) < 6 {
		return
	}

	gpsTime, err := parseGPSTime(dateStr, timeStr)
	if err != nil {
		return
	}

	g.mu.Lock()
	g.hasFix = true
	g.gpsTime = gpsTime
	g.mu.Unlock()
}

func (g *GPSReader) parseGGA(fields []string) {
	if len(fields) < 8 {
		return
	}

	fixQuality := fields[6]
	if fixQuality == "0" {
		return
	}

	sats := 0
	fmt.Sscanf(fields[7], "%d", &sats)

	g.mu.Lock()
	g.satellites = sats
	g.mu.Unlock()
}

func (g *GPSReader) GetState() (time.Time, bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.gpsTime, g.hasFix, g.satellites
}

func parseGPSTime(dateStr, timeStr string) (time.Time, error) {
	var day, month, year, hour, minute int
	var second float64

	fmt.Sscanf(dateStr[:2], "%d", &day)
	fmt.Sscanf(dateStr[2:4], "%d", &month)
	fmt.Sscanf(dateStr[4:6], "%d", &year)

	fmt.Sscanf(timeStr[:2], "%d", &hour)
	fmt.Sscanf(timeStr[2:4], "%d", &minute)
	fmt.Sscanf(timeStr[4:], "%f", &second)

	fullYear := 2000 + year
	if year > 80 {
		fullYear = 1900 + year
	}

	nsec := int((second - float64(int(second))) * 1e9)

	return time.Date(fullYear, time.Month(month), day, hour, minute, int(second), nsec, time.UTC), nil
}

type Broadcaster struct {
	broadcastAddr string
	interfaces    []string
	sourceID      string
	conn          *net.UDPConn
}

func NewBroadcaster(broadcastAddr, sourceID string, ifaces []string) (*Broadcaster, error) {
	addr, err := net.ResolveUDPAddr("udp4", broadcastAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	return &Broadcaster{
		broadcastAddr: broadcastAddr,
		sourceID:      sourceID,
		interfaces:    ifaces,
		conn:          conn,
	}, nil
}

func (b *Broadcaster) Broadcast(gpsTime time.Time, satellites int) error {
	now := time.Now()
	msg := &TimeAnnounceMessage{
		Type:      "TIME_ANNOUNCE",
		MessageID: fmt.Sprintf("%s-%d", b.sourceID, now.UnixNano()),
		Timestamp: gpsTime.UnixNano(),
		ClockInfo: ClockInfo{
			Stratum:        1,
			Precision:      -20,
			RootDelay:      0.0,
			RootDispersion: 0.0001,
			ReferenceID:    "GPS",
			ReferenceTime:  gpsTime.UnixNano(),
		},
		LeapIndicator: 0,
		SourceID:      b.sourceID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := b.conn.Write(data); err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	b.broadcastToInterfaces(data)

	return nil
}

func (b *Broadcaster) broadcastToInterfaces(data []byte) {
	_, port, err := net.SplitHostPort(b.broadcastAddr)
	if err != nil {
		return
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}

	ifaceFilter := make(map[string]bool)
	for _, name := range b.interfaces {
		ifaceFilter[name] = true
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if len(ifaceFilter) > 0 && !ifaceFilter[iface.Name] {
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
						b.conn.WriteTo(data, targetAddr)
					}
				}
			}
		}
	}
}

func (b *Broadcaster) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
}

func main() {
	flag.Parse()

	sourceID := *sourceID
	if sourceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "gps-broadcaster"
		}
		sourceID = hostname
	}

	var ifaces []string
	if *interfaces != "" {
		ifaces = strings.Split(*interfaces, ",")
	}

	if *verbose {
		fmt.Printf("GPS Broadcaster starting...\n")
		fmt.Printf("  Device: %s\n", *device)
		fmt.Printf("  Source ID: %s\n", sourceID)
		fmt.Printf("  Broadcast: %s\n", *broadcast)
		fmt.Printf("  Backoff: 10s -> 30s -> 2m -> 5m -> 10m -> 20m -> 30m -> 1h\n")
	}

	gps := NewGPSReader(*device, *baudRate)
	if err := gps.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening GPS device: %v\n", err)
		os.Exit(1)
	}
	defer gps.Close()

	broadcaster, err := NewBroadcaster(*broadcast, sourceID, ifaces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating broadcaster: %v\n", err)
		os.Exit(1)
	}
	defer broadcaster.Close()

	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		if *verbose {
			fmt.Println("\nShutting down...")
		}
		close(stopChan)
	}()

	backoff := NewBackoffManager()

	lastLoggedFix := false
	checkCount := 0

	serialTimeout := 500 * time.Millisecond
	readLoopInterval := 100 * time.Millisecond

	ticker := time.NewTicker(readLoopInterval)
	defer ticker.Stop()

	broadcastTimer := time.NewTimer(10 * time.Second)
	defer broadcastTimer.Stop()

	for {
		select {
		case <-stopChan:
			return

		case <-ticker.C:
			gps.SetReadDeadline(time.Now().Add(serialTimeout))
			line, err := gps.ReadLine()
			if err == nil {
				gps.ParseNMEA(line)
			}

		case <-broadcastTimer.C:
			checkCount++
			gpsTime, hasFix, sats := gps.GetState()

			if hasFix && !lastLoggedFix && *verbose {
				fmt.Printf("%s: GPS fix acquired (sats: %d)\n", time.Now().Format("15:04:05"), sats)
				lastLoggedFix = true
			} else if !hasFix && lastLoggedFix && *verbose {
				fmt.Printf("%s: GPS fix lost\n", time.Now().Format("15:04:05"))
				lastLoggedFix = false
			}

			if hasFix {
				if err := broadcaster.Broadcast(gpsTime, sats); err != nil {
					if *verbose {
						fmt.Printf("%s: Broadcast error: %v\n", time.Now().Format("15:04:05"), err)
					}
				} else if *verbose {
					offset := time.Since(gpsTime).Seconds()
					fmt.Printf("%s: Broadcast GPS time %s (offset: %.3fs, sats: %d)\n",
						time.Now().Format("15:04:05"),
						gpsTime.Format("2006-01-02 15:04:05 MST"),
						offset,
						sats)
				}
			}

			nextInterval := backoff.NextInterval(hasFix, sats)
			broadcastTimer.Reset(nextInterval)
		}
	}
}
