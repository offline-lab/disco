package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/offline-lab/disco/internal/config"
	"github.com/offline-lab/disco/internal/discovery"
	"github.com/offline-lab/disco/internal/security"
	"github.com/spf13/cobra"
)

var (
	listenBroadcastAddr string
	listenKeyFile       string
	listenRequireSigned bool
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for broadcast messages on the network",
	Long: `Listen for disco broadcast messages from all hosts on the network,
including this machine, and print them as they arrive.

No daemon connection is required — this binds directly to the broadcast
address and prints each message to stdout.

Examples:
  disco listen
  disco listen --broadcast 255.255.255.255:5354
  disco listen --json`,
	Args: cobra.NoArgs,
	RunE: runListen,
}

func init() {
	rootCmd.AddCommand(listenCmd)
	listenCmd.Flags().StringVarP(&listenBroadcastAddr, "broadcast", "b", "", "Broadcast address to listen on (default from config or 255.255.255.255:5354)")
	listenCmd.Flags().StringVarP(&listenKeyFile, "key-file", "k", "", "Key file for signature verification; unsigned messages are shown but flagged")
	listenCmd.Flags().BoolVar(&listenRequireSigned, "require-signed", false, "Drop unsigned or unverifiable messages (requires --key-file)")
}

func runListen(cmd *cobra.Command, args []string) error {
	if listenRequireSigned && listenKeyFile == "" {
		return fmt.Errorf("--require-signed requires --key-file")
	}

	addr := resolveBroadcastAddr()

	var km *security.KeyManager
	if listenKeyFile != "" {
		var err error
		km, err = security.LoadKeyManager(listenKeyFile)
		if err != nil {
			return err
		}
	}

	listener, err := discovery.NewListener(addr, km, listenRequireSigned)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Stop()

	stopChan := make(chan struct{})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	go func() {
		<-sigChan
		close(stopChan)
	}()

	if !outputJSON {
		fmt.Fprintf(os.Stderr, "Listening on %s — press Ctrl+C to stop\n\n", addr)
	}

	go listener.Start(stopChan)

	for {
		select {
		case <-stopChan:
			return nil
		case msg, ok := <-listener.Messages():
			if !ok {
				return nil
			}
			if outputJSON {
				printListenJSON(msg)
			} else {
				printListenText(msg)
			}
		}
	}
}

func resolveBroadcastAddr() string {
	if listenBroadcastAddr != "" {
		return listenBroadcastAddr
	}
	if configPath != "" {
		if cfg, err := config.Load(configPath); err == nil && cfg.Network.BroadcastAddr != "" {
			return cfg.Network.BroadcastAddr
		}
	}
	return "255.255.255.255:5354"
}

type listenEvent struct {
	Time      string              `json:"time"`
	Hostname  string              `json:"hostname"`
	MachineID string              `json:"machine_id,omitempty"`
	Addresses []string            `json:"addresses"`
	Services  []listenServiceInfo `json:"services,omitempty"`
	Signed    bool                `json:"signed"`
	Verified  *bool               `json:"verified,omitempty"` // nil when no key file loaded
}

type listenServiceInfo struct {
	Name    string   `json:"name"`
	Port    int      `json:"port"`
	Aliases []string `json:"aliases,omitempty"`
}

func printListenJSON(msg *discovery.BroadcastMessage) {
	svcs := make([]listenServiceInfo, 0, len(msg.Services))
	for _, s := range msg.Services {
		svcs = append(svcs, listenServiceInfo{Name: s.Name, Port: s.Port, Aliases: s.Aliases})
	}
	ev := listenEvent{
		Time:      time.Unix(msg.Timestamp, 0).Format(time.RFC3339),
		Hostname:  msg.Hostname,
		MachineID: msg.MachineID,
		Addresses: msg.IPs,
		Services:  svcs,
		Signed:    msg.Signature != nil,
	}
	// If a key file is loaded, any signed message that reaches the channel was
	// already verified by the listener (failures are dropped silently).
	if listenKeyFile != "" {
		v := msg.Signature != nil
		ev.Verified = &v
	}
	data, _ := json.Marshal(ev)
	fmt.Println(string(data))
}

func printListenText(msg *discovery.BroadcastMessage) {
	ts := time.Unix(msg.Timestamp, 0).Format("15:04:05")

	var sigTag string
	switch {
	case msg.Signature == nil:
		sigTag = "" // unsigned, no tag
	case listenKeyFile == "":
		sigTag = " [signed]" // signed but not verified (no key loaded)
	default:
		sigTag = " [verified]" // signed and reached the channel → verified by listener
	}

	id := ""
	if msg.MachineID != "" {
		short := msg.MachineID
		if len(short) > 8 {
			short = short[:8]
		}
		id = fmt.Sprintf(" (%s)", short)
	}

	fmt.Printf("%s  %-20s%s%s\n", ts, msg.Hostname+id, formatAddrs(msg.IPs), sigTag)

	for _, svc := range msg.Services {
		fmt.Printf("         └─ %-14s port %d\n", svc.Name, svc.Port)
	}
}

func formatAddrs(ips []string) string {
	if len(ips) == 0 {
		return ""
	}
	out := "  "
	for i, ip := range ips {
		if i > 0 {
			out += ", "
		}
		out += ip
	}
	return out
}
