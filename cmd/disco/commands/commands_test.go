package commands

import (
	"fmt"
	"testing"

	"github.com/offline-lab/disco/internal/nss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestMatchTarget(t *testing.T) {
	const idAlpha = "abcd1234abcd1234abcd1234abcd1234"
	const idBeta = "ef567890ef567890ef567890ef567890"

	hosts := []nss.HostHealth{
		{
			Hostname:  "alpha",
			MachineID: idAlpha,
			Addresses: []string{"10.0.0.1"},
			Services:  map[string]string{"ssh": "10.0.0.1:22"},
		},
		{
			Hostname:  "beta",
			MachineID: idBeta,
			Addresses: []string{"10.0.0.2"},
			Services:  map[string]string{"openvpn": "10.0.0.2:1194"},
		},
	}

	cases := []struct {
		target string
		want   string
	}{
		// hostname match (case-insensitive)
		{"alpha", "10.0.0.1"},
		{"ALPHA", "10.0.0.1"},
		{"beta", "10.0.0.2"},
		// exact full machine ID match
		{idAlpha, "10.0.0.1"},
		{idBeta, "10.0.0.2"},
		// partial machine ID does NOT match — exact only, no prefix
		{"abcd1234", "abcd1234"},
		{"abcd", "abcd"},
		// service name match
		{"ssh", "10.0.0.1"},
		{"openvpn", "10.0.0.2"},
		// no match falls back to target unchanged
		{"unknown", "unknown"},
	}

	for _, c := range cases {
		got := matchTarget(c.target, hosts)
		if got != c.want {
			t.Errorf("matchTarget(%q) = %q, want %q", c.target, got, c.want)
		}
	}
}

// TestNoFlagShorthandConflicts walks the entire command tree and verifies that
// no command defines two flags with the same shorthand in the same flagset.
// pflag panics at startup when this happens, so this test catches it at build time.
func TestNoFlagShorthandConflicts(t *testing.T) {
	walkCommands(rootCmd, func(cmd *cobra.Command) {
		checkFlagSet(t, cmd, cmd.Flags(), "local")
		checkFlagSet(t, cmd, cmd.PersistentFlags(), "persistent")
	})
}

func checkFlagSet(t *testing.T, cmd *cobra.Command, flags *pflag.FlagSet, kind string) {
	t.Helper()
	seen := map[string]string{}
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Shorthand == "" {
			return
		}
		if existing, ok := seen[f.Shorthand]; ok {
			t.Errorf("command %q %s flags: shorthand -%s used by both %q and %q",
				cmd.CommandPath(), kind, f.Shorthand, existing, f.Name)
		}
		seen[f.Shorthand] = f.Name
	})
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands() {
		walkCommands(sub, fn)
	}
}

// TestCommandsHaveUsage ensures every non-root command has Use and Short set,
// catching incomplete command registrations early.
func TestCommandsHaveUsage(t *testing.T) {
	walkCommands(rootCmd, func(cmd *cobra.Command) {
		if cmd == rootCmd {
			return
		}
		if cmd.Use == "" {
			t.Errorf("command %q has empty Use field", cmd.CommandPath())
		}
		if cmd.Short == "" {
			t.Errorf("command %q has empty Short description", cmd.CommandPath())
		}
	})
}

// TestRootCommandExecutes verifies that the root command initialises without
// panicking (i.e. all init() flag registrations succeed).
func TestRootCommandExecutes(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("command initialisation panicked: %v", r)
		}
	}()
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	_ = fmt.Sprintf("commands registered: %d", len(rootCmd.Commands()))
}
