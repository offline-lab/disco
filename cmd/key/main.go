package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

const keySize = 32

type Keys struct {
	PublicKey  string   `json:"public_key"`
	PrivateKey string   `json:"private_key"`
	PublicKeys []string `json:"public_keys"`
}

func printHelp() {
	fmt.Println("nss-key - Security key management for NSS daemon")
	fmt.Println()
	fmt.Println("WHAT:")
	fmt.Println("  Generate, view, and manage cryptographic keys for securing")
	fmt.Println("  broadcast messages between nodes.")
	fmt.Println()
	fmt.Println("WHY:")
	fmt.Println("  • Secure discovery against spoofing and replay attacks")
	fmt.Println("  • Only accept messages from trusted peers")
	fmt.Println("  • HMAC-SHA256 signatures with replay protection")
	fmt.Println("  • Optional - disable for maximum speed on trusted networks")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  nss-key <command> [args]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  generate, gen   Generate a new key pair")
	fmt.Println("  show           Display current keys and trusted peers")
	fmt.Println("  add-trusted    Add a trusted peer's public key")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  generate:")
	fmt.Println("    [key-file]    Path for keys file (default: /etc/nss-daemon/keys.json)")
	fmt.Println("  show:")
	fmt.Println("    [key-file]    Path to keys file")
	fmt.Println("  add-trusted:")
	fmt.Println("    <public-key>  The peer's public key (64 hex chars)")
	fmt.Println("    [key-file]    Path to keys file")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  nss-key generate")
	fmt.Println("  nss-key show")
	fmt.Println("  nss-key add-trusted abc123def456...")
	fmt.Println("  nss-key generate /custom/keys.json")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/offline-lab/disco")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate", "gen":
		generateKeys()
	case "show":
		showKeys()
	case "add-trusted":
		addTrustedKey()
	case "help", "--help", "-h":
		printHelp()
	default:
		printHelp()
		os.Exit(1)
	}
}

func generateKeys() {
	keyPath := "/etc/nss-daemon/keys.json"
	if len(os.Args) > 2 {
		keyPath = os.Args[2]
	}

	privateKey := make([]byte, keySize)
	publicKey := make([]byte, keySize)

	if _, err := rand.Read(privateKey); err != nil {
		fmt.Printf("Error generating private key: %v\n", err)
		os.Exit(1)
	}

	if _, err := rand.Read(publicKey); err != nil {
		fmt.Printf("Error generating public key: %v\n", err)
		os.Exit(1)
	}

	keys := Keys{
		PrivateKey: hex.EncodeToString(privateKey),
		PublicKey:  hex.EncodeToString(publicKey),
		PublicKeys: []string{hex.EncodeToString(publicKey)},
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling keys: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(keyPath, data, 0600); err != nil {
		fmt.Printf("Error writing keys to %s: %v\n", keyPath, err)
		os.Exit(1)
	}

	fmt.Printf("Keys generated and saved to %s\n", keyPath)
	fmt.Printf("Public Key: %s\n", hex.EncodeToString(publicKey))
	fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey))
	fmt.Println()
	fmt.Println("Share the public key with peers you want to trust.")
	fmt.Println("Keep the private key secret!")
}

func showKeys() {
	keyPath := "/etc/nss-daemon/keys.json"
	if len(os.Args) > 2 {
		keyPath = os.Args[2]
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("Error reading keys from %s: %v\n", keyPath, err)
		os.Exit(1)
	}

	var keys Keys
	if err := json.Unmarshal(data, &keys); err != nil {
		fmt.Printf("Error parsing keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Public Key:")
	fmt.Println(keys.PublicKey)
	fmt.Println()
	fmt.Println("Private Key:")
	fmt.Println(keys.PrivateKey)
	fmt.Println()
	fmt.Printf("Trusted Peers (%d):\n", len(keys.PublicKeys))
	for i, pubKey := range keys.PublicKeys {
		fmt.Printf("  %d: %s\n", i+1, pubKey)
	}
}

func addTrustedKey() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: nss-key add-trusted <public-key> [key-file]")
		fmt.Println()
		fmt.Println("Example: nss-key add-trusted abc123def456...")
		os.Exit(1)
	}

	pubKey := os.Args[2]
	keyPath := "/etc/nss-daemon/keys.json"
	if len(os.Args) > 3 {
		keyPath = os.Args[3]
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("Error reading keys from %s: %v\n", keyPath, err)
		os.Exit(1)
	}

	var keys Keys
	if err := json.Unmarshal(data, &keys); err != nil {
		fmt.Printf("Error parsing keys: %v\n", err)
		os.Exit(1)
	}

	for _, existing := range keys.PublicKeys {
		if existing == pubKey {
			fmt.Println("Public key already in trusted peers list")
			return
		}
	}

	keys.PublicKeys = append(keys.PublicKeys, pubKey)

	newData, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling keys: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(keyPath, newData, 0600); err != nil {
		fmt.Printf("Error writing keys to %s: %v\n", keyPath, err)
		os.Exit(1)
	}

	fmt.Println("Added trusted peer:")
	fmt.Println(pubKey)
	fmt.Println()
	fmt.Printf("Total trusted peers: %d\n", len(keys.PublicKeys))
}
