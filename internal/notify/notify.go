package notify

import (
	"net"
	"os"
	"strings"
)

// Ready sends READY=1 to systemd's $NOTIFY_SOCKET if set. No-op if not running under systemd.
func Ready() {
	send("READY=1\n")
}

// Stopping sends STOPPING=1 to systemd's $NOTIFY_SOCKET if set.
// Call this before beginning graceful shutdown so systemd does not treat the exit as a crash.
func Stopping() {
	send("STOPPING=1\n")
}

func send(payload string) {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return
	}

	// Abstract namespace sockets start with '@'
	if strings.HasPrefix(socket, "@") {
		socket = "\x00" + socket[1:]
	}

	conn, err := net.Dial("unixgram", socket)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	_, _ = conn.Write([]byte(payload))
}
