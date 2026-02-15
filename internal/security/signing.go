package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	keySize   = 32
	nonceSize = 16
	replayTTL = 300 // 5 minutes in seconds
)

// MessageSecurity holds security information for broadcast messages
type MessageSecurity struct {
	Signature string `json:"signature,omitempty"`
	Signer    string `json:"signer,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// KeyManager manages security keys
type KeyManager struct {
	privateKey string
	publicKey  string
	publicKeys map[string]string
	mu         sync.RWMutex
	keyPath    string
}

// NewKeyManager creates a new key manager
func NewKeyManager(keyPath string) (*KeyManager, error) {
	km := &KeyManager{
		publicKeys: make(map[string]string),
		keyPath:    keyPath,
	}

	if err := km.loadOrGenerateKeys(); err != nil {
		return nil, fmt.Errorf("failed to setup keys: %w", err)
	}

	return km, nil
}

// loadOrGenerateKeys loads existing keys or generates new ones
func (km *KeyManager) loadOrGenerateKeys() error {
	if km.keyPath == "" {
		return km.generateKeys()
	}

	data, err := os.ReadFile(km.keyPath)
	if err != nil {
		return km.generateKeys()
	}

	var keys struct {
		PublicKey  string   `json:"public_key"`
		PrivateKey string   `json:"private_key"`
		PublicKeys []string `json:"public_keys"`
	}

	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to parse keys file: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	km.privateKey = keys.PrivateKey
	km.publicKey = keys.PublicKey

	for _, pubKey := range keys.PublicKeys {
		km.publicKeys[pubKey] = pubKey
	}

	return nil
}

// generateKeys generates a new key pair
func (km *KeyManager) generateKeys() error {
	privateKey := make([]byte, keySize)
	publicKey := make([]byte, keySize)

	if _, err := rand.Read(privateKey); err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	if _, err := rand.Read(publicKey); err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	km.privateKey = hex.EncodeToString(privateKey)
	km.publicKey = hex.EncodeToString(publicKey)

	return nil
}

// Sign signs a message using HMAC-SHA256
func (km *KeyManager) Sign(message []byte) (*MessageSecurity, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create HMAC signature
	h := sha256.New()
	h.Write([]byte(km.privateKey))
	h.Write([]byte(nonce))
	h.Write(message)
	signature := hex.EncodeToString(h.Sum(nil))

	return &MessageSecurity{
		Signature: signature,
		Signer:    km.publicKey,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
	}, nil
}

// Verify verifies a message signature
func (km *KeyManager) Verify(message []byte, ms *MessageSecurity) bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	verificationKey, known := km.publicKeys[ms.Signer]
	if !known {
		return false
	}

	if time.Now().Unix()-ms.Timestamp > replayTTL {
		return false
	}

	h := sha256.New()
	h.Write([]byte(verificationKey))
	h.Write([]byte(ms.Nonce))
	h.Write(message)
	expectedSig := hex.EncodeToString(h.Sum(nil))

	return ms.Signature == expectedSig
}

// ValidateMessage validates a broadcast message
func (km *KeyManager) ValidateMessage(msg interface{}) error {
	// Check for required security fields
	msgMap, ok := msg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("message is not a map")
	}

	// Check timestamp is recent enough
	timestamp, ok := msgMap["timestamp"].(float64)
	if ok {
		if time.Now().Unix()-int64(timestamp) > replayTTL {
			return fmt.Errorf("message timestamp too old")
		}
	}

	// Check signature if security is enabled
	signature, ok := msgMap["signature"].(string)
	if ok && signature != "" {
		if _, ok := msgMap["signer"].(string); ok {
			if _, ok := msgMap["nonce"].(string); ok {
				// Validate signature
				// This is a placeholder - full signature validation
				// would require the full message content
			}
		}
	}

	return nil
}

// AddTrustedPeer adds a trusted peer's verification key (shared secret)
// In this shared-secret HMAC system, the verificationKey is the peer's privateKey
func (km *KeyManager) AddTrustedPeer(hostname string, verificationKey string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	km.publicKeys[verificationKey] = verificationKey
}

// AddTrustedPeerByID adds a trusted peer using their publicKey as identifier and privateKey as verification key
func (km *KeyManager) AddTrustedPeerByID(publicKey string, privateKey string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	km.publicKeys[publicKey] = privateKey
}

// GetPrivateKey returns the private key (shared secret)
func (km *KeyManager) GetPrivateKey() string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	return km.privateKey
}

// GetPublicKey returns the public key
func (km *KeyManager) GetPublicKey() string {
	km.mu.RLock()
	defer km.mu.RUnlock()

	return km.publicKey
}

// SaveKeys saves the current keys to a file
func (km *KeyManager) SaveKeys(path string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var keys struct {
		PublicKey  string   `json:"public_key"`
		PrivateKey string   `json:"private_key"`
		PublicKeys []string `json:"public_keys"`
	}

	keys.PublicKey = km.publicKey
	keys.PrivateKey = km.privateKey

	for _, pubKey := range km.publicKeys {
		keys.PublicKeys = append(keys.PublicKeys, pubKey)
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// generateNonce generates a random nonce
func generateNonce() (string, error) {
	nonce := make([]byte, nonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}
