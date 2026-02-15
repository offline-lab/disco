package security

import (
	"testing"
)

func TestKeyManager_New(t *testing.T) {
	km, err := NewKeyManager("/tmp/test-key.json")
	if err != nil {
		t.Fatalf("NewKeyManager() error = %v", err)
	}

	if km == nil {
		t.Fatal("NewKeyManager() returned nil")
	}

	if km.GetPublicKey() == "" {
		t.Error("GetPublicKey() returned empty string")
	}
}

func TestKeyManager_Sign(t *testing.T) {
	km, _ := NewKeyManager("/tmp/test-key.json")
	if km == nil {
		t.Fatal("Could not create key manager")
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
	km, _ := NewKeyManager("/tmp/test-key.json")
	if km == nil {
		t.Fatal("Could not create key manager")
	}

	message := []byte("test message")
	sig, _ := km.Sign(message)

	km.AddTrustedPeerByID(km.GetPublicKey(), km.GetPrivateKey())

	if !km.Verify(message, sig) {
		t.Error("Verify() returned false for valid signature")
	}

	wrongMessage := []byte("wrong message")
	if km.Verify(wrongMessage, sig) {
		t.Error("Verify() returned true for wrong message")
	}

	km2, _ := NewKeyManager("/tmp/test-key2.json")
	sig2, _ := km2.Sign(message)
	if km.Verify(message, sig2) {
		t.Error("Verify() returned true for unknown signer")
	}
}

func TestKeyManager_AddTrustedPeer(t *testing.T) {
	km, _ := NewKeyManager("/tmp/test-key.json")
	if km == nil {
		t.Fatal("Could not create key manager")
	}

	message := []byte("test message")
	sig, _ := km.Sign(message)

	km.AddTrustedPeerByID(km.GetPublicKey(), km.GetPrivateKey())

	if !km.Verify(message, sig) {
		t.Error("Verify() failed after adding self as trusted peer")
	}
}

func TestKeyManager_SaveKeys(t *testing.T) {
	km, _ := NewKeyManager("/tmp/test-key-save.json")
	if km == nil {
		t.Fatal("Could not create key manager")
	}

	err := km.SaveKeys("/tmp/test-key-saved.json")
	if err != nil {
		t.Fatalf("SaveKeys() error = %v", err)
	}
}

func TestMessageSecurity_Validation(t *testing.T) {
	km, _ := NewKeyManager("/tmp/test-key.json")
	if km == nil {
		t.Fatal("Could not create key manager")
	}

	message := []byte("test message")
	sig, _ := km.Sign(message)

	// Create a message with security info
	msgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": sig.Timestamp,
		"signature": sig.Signature,
		"signer":    sig.Signer,
		"nonce":     sig.Nonce,
	}

	err := km.ValidateMessage(msgMap)
	if err != nil {
		t.Errorf("ValidateMessage() error = %v", err)
	}

	// Test old timestamp
	oldMsgMap := map[string]interface{}{
		"type":      "announce",
		"hostname":  "test-host",
		"timestamp": float64(0), // Very old timestamp
	}

	err = km.ValidateMessage(oldMsgMap)
	if err == nil {
		t.Error("ValidateMessage() should reject old timestamp")
	}
}
