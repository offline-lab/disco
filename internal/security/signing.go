package security

import (
	"crypto/hmac"
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
	replayTTL = 300
)

type MessageSecurity struct {
	Signature string `json:"signature,omitempty"`
	Signer    string `json:"signer,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type KeyManager struct {
	sharedSecret string
	nodeID       string
	trustedPeers map[string]string
	mu           sync.RWMutex
	keyPath      string
}

func NewKeyManager(keyPath string) (*KeyManager, error) {
	km := &KeyManager{
		trustedPeers: make(map[string]string),
		keyPath:      keyPath,
	}

	if err := km.loadOrGenerateKeys(); err != nil {
		return nil, fmt.Errorf("failed to setup keys: %w", err)
	}

	return km, nil
}

func (km *KeyManager) loadOrGenerateKeys() error {
	if km.keyPath == "" {
		return km.generateKeys()
	}

	data, err := os.ReadFile(km.keyPath)
	if err != nil {
		return km.generateKeys()
	}

	var keys struct {
		PublicKey    string            `json:"public_key"`
		PrivateKey   string            `json:"private_key"`
		SharedSecret string            `json:"shared_secret"`
		NodeID       string            `json:"node_id"`
		TrustedPeers map[string]string `json:"trusted_peers"`
		PublicKeys   []string          `json:"public_keys"`
	}

	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to parse keys file: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	if keys.SharedSecret != "" {
		km.sharedSecret = keys.SharedSecret
		km.nodeID = keys.NodeID
		for nodeID, secret := range keys.TrustedPeers {
			km.trustedPeers[nodeID] = secret
		}
	} else {
		km.sharedSecret = keys.PrivateKey
		km.nodeID = keys.PublicKey
		for _, pubKey := range keys.PublicKeys {
			km.trustedPeers[pubKey] = keys.PrivateKey
		}
	}

	return nil
}

func (km *KeyManager) generateKeys() error {
	sharedSecret := make([]byte, keySize)
	nodeID := make([]byte, keySize)

	if _, err := rand.Read(sharedSecret); err != nil {
		return fmt.Errorf("failed to generate shared secret: %w", err)
	}

	if _, err := rand.Read(nodeID); err != nil {
		return fmt.Errorf("failed to generate node ID: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	km.sharedSecret = hex.EncodeToString(sharedSecret)
	km.nodeID = hex.EncodeToString(nodeID)
	km.trustedPeers[km.nodeID] = km.sharedSecret

	return nil
}

func (km *KeyManager) Sign(message []byte) (*MessageSecurity, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(km.sharedSecret))
	mac.Write([]byte(nonce))
	mac.Write(message)
	signature := hex.EncodeToString(mac.Sum(nil))

	return &MessageSecurity{
		Signature: signature,
		Signer:    km.nodeID,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (km *KeyManager) Verify(message []byte, ms *MessageSecurity) bool {
	if ms == nil {
		return false
	}

	km.mu.RLock()
	defer km.mu.RUnlock()

	peerSecret, known := km.trustedPeers[ms.Signer]
	if !known {
		return false
	}

	if time.Now().Unix()-ms.Timestamp > replayTTL {
		return false
	}
	if time.Now().Unix() < ms.Timestamp {
		return false
	}

	mac := hmac.New(sha256.New, []byte(peerSecret))
	mac.Write([]byte(ms.Nonce))
	mac.Write(message)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(ms.Signature), []byte(expectedSig))
}

func (km *KeyManager) ValidateMessage(msg interface{}) error {
	msgMap, ok := msg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("message is not a map")
	}

	timestamp, ok := msgMap["timestamp"].(float64)
	if ok {
		ts := int64(timestamp)
		now := time.Now().Unix()
		if now-ts > replayTTL {
			return fmt.Errorf("message timestamp too old")
		}
		if ts > now+60 {
			return fmt.Errorf("message timestamp in future")
		}
	}

	signature, _ := msgMap["signature"].(string)
	signer, _ := msgMap["signer"].(string)
	nonce, _ := msgMap["nonce"].(string)

	if signature != "" && signer != "" && nonce != "" {
		km.mu.RLock()
		_, known := km.trustedPeers[signer]
		km.mu.RUnlock()
		if !known {
			return fmt.Errorf("unknown signer: %s", signer)
		}
	}

	return nil
}

func (km *KeyManager) AddTrustedPeer(hostname string, sharedSecret string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.trustedPeers[hostname] = sharedSecret
}

func (km *KeyManager) AddTrustedPeerByID(nodeID string, sharedSecret string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.trustedPeers[nodeID] = sharedSecret
}

func (km *KeyManager) GetPrivateKey() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.sharedSecret
}

func (km *KeyManager) GetPublicKey() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.nodeID
}

func (km *KeyManager) GetNodeID() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.nodeID
}

func (km *KeyManager) GetSharedSecret() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.sharedSecret
}

func (km *KeyManager) SaveKeys(path string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys := struct {
		SharedSecret string            `json:"shared_secret"`
		NodeID       string            `json:"node_id"`
		TrustedPeers map[string]string `json:"trusted_peers"`
		PublicKey    string            `json:"public_key,omitempty"`
		PrivateKey   string            `json:"private_key,omitempty"`
		PublicKeys   []string          `json:"public_keys,omitempty"`
	}{
		SharedSecret: km.sharedSecret,
		NodeID:       km.nodeID,
		TrustedPeers: km.trustedPeers,
		PublicKey:    km.nodeID,
		PrivateKey:   km.sharedSecret,
	}

	for nodeID := range km.trustedPeers {
		keys.PublicKeys = append(keys.PublicKeys, nodeID)
	}

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func generateNonce() (string, error) {
	nonce := make([]byte, nonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}
