package commands

import (
	"strings"
	"testing"
)

func TestFormatAddrs(t *testing.T) {
	cases := []struct {
		ips  []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"10.0.0.1"}, "  10.0.0.1"},
		{[]string{"10.0.0.1", "10.0.0.2"}, "  10.0.0.1, 10.0.0.2"},
	}

	for _, c := range cases {
		got := formatAddrs(c.ips)
		if got != c.want {
			t.Errorf("formatAddrs(%v) = %q, want %q", c.ips, got, c.want)
		}
	}
}

func TestResolveBroadcastAddr(t *testing.T) {
	orig := listenBroadcastAddr
	origConfig := configPath
	defer func() {
		listenBroadcastAddr = orig
		configPath = origConfig
	}()

	// Flag takes priority.
	listenBroadcastAddr = "192.168.1.255:9000"
	configPath = ""
	got := resolveBroadcastAddr()
	if got != "192.168.1.255:9000" {
		t.Errorf("resolveBroadcastAddr() = %q, want explicit flag value", got)
	}

	// Default when nothing is set.
	listenBroadcastAddr = ""
	configPath = ""
	got = resolveBroadcastAddr()
	if got != "255.255.255.255:5354" {
		t.Errorf("resolveBroadcastAddr() = %q, want default 255.255.255.255:5354", got)
	}
}

func TestRunListen_RequireSignedWithoutKeyFile(t *testing.T) {
	orig := listenRequireSigned
	origKey := listenKeyFile
	defer func() {
		listenRequireSigned = orig
		listenKeyFile = origKey
	}()

	listenRequireSigned = true
	listenKeyFile = ""

	err := runListen(listenCmd, nil)
	if err == nil {
		t.Fatal("runListen() should return error when --require-signed is set without --key-file")
	}
	if !strings.Contains(err.Error(), "--require-signed requires --key-file") {
		t.Errorf("unexpected error: %v", err)
	}
}
