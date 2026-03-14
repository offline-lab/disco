package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyManager_New(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	if km == nil {
		t.Fatal("NewKeyManager() returned nil")
	}

	if km.GetNodeID() == "" {
		t.Error("GetNodeID() returned empty string")
	}

	if km.GetSharedSecret() == "" {
		t.Error("GetSharedSecret() returned empty string")
	}
}

func TestKeyManager_Sign(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	message := []byte("test message")
	sig, err := km.Sign(message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if sig == nil {
		t.Fatal("Sign() returned nil signature")
	}

	if sig.Signature == "" {
		t.Error("Signature is empty")
	}

	if sig.Signer == "" {
		t.Error("Signer is empty")
	}

	if sig.Nonce == "" {
		t.Error("Nonce is empty")
	}

	if sig.Timestamp == 0 {
		t.Error("Timestamp is zero")
	}
}

func TestKeyManager_Verify(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	message := []byte("test message")
	sig, err := km.Sign(message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if !km.Verify(message, sig) {
		t.Error("Verify() returned false for valid signature")
	}

	wrongMessage := []byte("wrong message")
	if km.Verify(wrongMessage, sig) {
		t.Error("Verify() returned true for wrong message")
	}

	km2, _ := NewKeyManager(filepath.Join(t.TempDir(), "test-key2.json"))
	sig2, _ := km2.Sign(message)
	if km.Verify(message, sig2) {
		t.Error("Verify() returned true for unknown signer")
	}
}

func TestKeyManager_Verify_NilSignature(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	if km.Verify([]byte("test"), nil) {
		t.Error("Verify() returned true for nil signature")
	}
}

func TestKeyManager_Verify_FutureTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	sig := &MessageSecurity{
		Signature: "abc",
		Signer:    km.GetNodeID(),
		Nonce:     "def",
		Timestamp: 9999999999,
	}

	if km.Verify([]byte("test"), sig) {
		t.Error("Verify() returned true for future timestamp")
	}
}

func TestKeyManager_AddTrustedPeer(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	message := []byte("test message")
	sig, err := km.Sign(message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if !km.Verify(message, sig) {
		t.Error("Verify() failed for self-signed message")
	}
}

func TestKeyManager_SaveKeys(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key-save.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	savePath := filepath.Join(tmpDir, "test-key-saved.json")
	err = km.SaveKeys(savePath)
	if err != nil {
		t.Fatalf("SaveKeys() error = %v", err)
	}

	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Error("SaveKeys() did not create file")
	}

	loaded, err := NewKeyManager(savePath)
	if err != nil {
		t.Fatalf("Failed to load saved keys: %v", err)
	}

	if loaded.GetNodeID() != km.GetNodeID() {
		t.Error("NodeID mismatch after reload")
	}

	if loaded.GetSharedSecret() != km.GetSharedSecret() {
		t.Error("SharedSecret mismatch after reload")
	}
}

func TestKeyManager_ValidateMessage(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	sig, _ := km.Sign([]byte("test"))

	msgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": float64(sig.Timestamp),
		"signature": sig.Signature,
		"signer":    sig.Signer,
		"nonce":     sig.Nonce,
	}

	err = km.ValidateMessage(msgMap)
	if err != nil {
		t.Errorf("ValidateMessage() error = %v", err)
	}
}

func TestKeyManager_ValidateMessage_OldTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	oldMsgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": float64(0),
	}

	err = km.ValidateMessage(oldMsgMap)
	if err == nil {
		t.Error("ValidateMessage() should reject old timestamp")
	}
}

func TestKeyManager_ValidateMessage_FutureTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	futureMsgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": float64(9999999999),
	}

	err = km.ValidateMessage(futureMsgMap)
	if err == nil {
		t.Error("ValidateMessage() should reject future timestamp")
	}
}

func TestKeyManager_ValidateMessage_UnknownSigner(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	sig, _ := km.Sign([]byte("test"))
	msgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": float64(sig.Timestamp),
		"signature": "somesig",
		"signer":    "unknown-signer",
		"nonce":     "somenonce",
	}

	err = km.ValidateMessage(msgMap)
	if err == nil {
		t.Error("ValidateMessage() should reject unknown signer")
	}
}

func TestKeyManager_ValidateMessage_NotAMap(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-key.json")

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	err = km.ValidateMessage("not a map")
	if err == nil {
		t.Error("ValidateMessage() should reject non-map input")
	}
}

func TestKeyManager_BackwardsCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "legacy-key.json")

	legacyFormat := `{
		"public_key": "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
		"private_key": "2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40",
		"public_keys": ["0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"]
	}`

	if err := os.WriteFile(keyPath, []byte(legacyFormat), 0600); err != nil {
		t.Fatalf("Failed to write legacy key file: %v", err)
	}

	km, err := NewKeyManager(keyPath)
	if err != nil {
		t.Fatalf("NewKeyManager() error loading legacy format: %v", err)
	}

	if km.GetPublicKey() != "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" {
		t.Error("Legacy public_key not loaded correctly")
	}

	if km.GetPrivateKey() != "2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" {
		t.Error("Legacy private_key not loaded correctly")
	}
}
