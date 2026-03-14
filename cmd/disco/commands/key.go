package commands

import (
	"fmt"

	"github.com/offline-lab/disco/cmd/disco/internal/cli"
	"github.com/offline-lab/disco/cmd/disco/internal/keys"
	"github.com/spf13/cobra"
)

const DefaultKeysPath = "/etc/disco/keys.json"

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage cryptographic keys",
	Long: `Manage cryptographic keys for securing broadcast messages.

Keys are used to sign and verify messages between nodes.
The private key must be kept secret, while the public key can be shared with trusted peers.`,
}

var keyGenerateCmd = &cobra.Command{
	Use:   "generate [key-file]",
	Short: "Generate a new key pair",
	Long:  "Generate a new public/private key pair and save to the specified file (default: /etc/disco/keys.json)",
	Args:  cobra.MaximumNArgs(1),
	Run:   generateKeys,
}

var keyShowCmd = &cobra.Command{
	Use:   "show [key-file]",
	Short: "Display current keys",
	Long:  "Display the current public/private keys and list of trusted peers from the specified file",
	Args:  cobra.MaximumNArgs(1),
	Run:   showKeys,
}

var keyAddTrustedCmd = &cobra.Command{
	Use:   "add-trusted <public-key> [key-file]",
	Short: "Add a trusted peer's public key",
	Long:  "Add a peer's public key to the trusted peers list (64 hex characters)",
	Args:  cobra.MinimumNArgs(1),
	Run:   addTrustedKey,
}

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.AddCommand(keyGenerateCmd)
	keyCmd.AddCommand(keyShowCmd)
	keyCmd.AddCommand(keyAddTrustedCmd)
}

func generateKeys(cmd *cobra.Command, args []string) {
	keyPath := DefaultKeysPath
	if len(args) > 0 {
		keyPath = args[0]
	}

	k, err := keys.Generate()
	if err != nil {
		checkError(fmt.Errorf("failed to generate keys: %w", err))
	}

	if err := k.Save(keyPath); err != nil {
		checkError(fmt.Errorf("failed to save keys: %w", err))
	}

	fmt.Printf("Keys generated and saved to %s\n", keyPath)
	fmt.Printf("Public Key: %s\n", k.PublicKey)
	fmt.Printf("Private Key: %s\n", k.PrivateKey)
	fmt.Println()
	fmt.Println("Share the public key with peers you want to trust.")
	fmt.Println("Keep the private key secret!")
}

func showKeys(cmd *cobra.Command, args []string) {
	keyPath := DefaultKeysPath
	if len(args) > 0 {
		keyPath = args[0]
	}

	k, err := keys.Load(keyPath)
	if err != nil {
		checkError(fmt.Errorf("failed to load keys: %w", err))
	}

	fmt.Println("Public Key:")
	fmt.Println(k.PublicKey)
	fmt.Println()
	fmt.Println("Private Key:")
	fmt.Println(k.PrivateKey)
	fmt.Println()
	fmt.Printf("Trusted Peers (%d):\n", len(k.PublicKeys))
	for i, pubKey := range k.PublicKeys {
		fmt.Printf("  %d: %s\n", i+1, pubKey)
	}
}

func addTrustedKey(cmd *cobra.Command, args []string) {
	pubKey := args[0]

	if err := cli.ValidateHexKey(pubKey, keys.HexKeyLength); err != nil {
		checkError(fmt.Errorf("invalid public key: %w", err))
	}

	keyPath := DefaultKeysPath
	if len(args) > 1 {
		keyPath = args[1]
	}

	k, err := keys.Load(keyPath)
	if err != nil {
		checkError(fmt.Errorf("failed to load keys: %w", err))
	}

	if err := k.AddTrustedPeer(pubKey); err != nil {
		if err.Error() == "public key already in trusted peers list" {
			fmt.Println("Public key already in trusted peers list")
			return
		}
		checkError(fmt.Errorf("failed to add trusted peer: %w", err))
	}

	if err := k.Save(keyPath); err != nil {
		checkError(fmt.Errorf("failed to save keys: %w", err))
	}

	fmt.Println("Added trusted peer:")
	fmt.Println(pubKey)
	fmt.Println()
	fmt.Printf("Total trusted peers: %d\n", len(k.PublicKeys))
}
