package keys

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const KeySize = 32

type Keys struct {
	PublicKey  string   `json:"public_key"`
	PrivateKey string   `json:"private_key"`
	PublicKeys []string `json:"public_keys"`
}

func Generate() (*Keys, error) {
	privateKey, err := generateSecureRandom(KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey, err := generateSecureRandom(KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	keys := &Keys{
		PrivateKey: hex.EncodeToString(privateKey),
		PublicKey:  hex.EncodeToString(publicKey),
		PublicKeys: []string{hex.EncodeToString(publicKey)},
	}

	return keys, nil
}

func generateSecureRandom(n int) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func Load(path string) (*Keys, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var keys Keys
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse keys: %w", err)
	}

	if err := keys.Validate(); err != nil {
		return nil, fmt.Errorf("invalid keys: %w", err)
	}

	return &keys, nil
}

func (k *Keys) Save(path string) error {
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write keys: %w", err)
	}

	return nil
}

func (k *Keys) Validate() error {
	if err := validateHexKey(k.PrivateKey, HexKeyLength); err != nil {
		return fmt.Errorf("private key: %w", err)
	}

	if err := validateHexKey(k.PublicKey, HexKeyLength); err != nil {
		return fmt.Errorf("public key: %w", err)
	}

	for i, pubKey := range k.PublicKeys {
		if err := validateHexKey(pubKey, HexKeyLength); err != nil {
			return fmt.Errorf("trusted peer %d: %w", i, err)
		}
	}

	return nil
}

func (k *Keys) AddTrustedPeer(pubKey string) error {
	if err := validateHexKey(pubKey, HexKeyLength); err != nil {
		return err
	}

	for _, existing := range k.PublicKeys {
		if existing == pubKey {
			return fmt.Errorf("public key already in trusted peers list")
		}
	}

	k.PublicKeys = append(k.PublicKeys, pubKey)
	return nil
}

const HexKeyLength = 64

func validateHexKey(key string, expectedLen int) error {
	if len(key) != expectedLen {
		return fmt.Errorf("must be %d hex characters, got %d", expectedLen, len(key))
	}

	for _, c := range key {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return fmt.Errorf("must be hexadecimal")
		}
	}

	return nil
}
