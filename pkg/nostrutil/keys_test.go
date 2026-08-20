package nostrutil

import (
	"testing"
)

func TestNostrKeys(t *testing.T) {
	// Sample test hex private key (32 bytes)
	testHexPriv := "0000000000000000000000000000000000000000000000000000000000000001"

	// Test ParsePrivateKey with hex
	hexKey, err := ParsePrivateKey(testHexPriv)
	if err != nil {
		t.Fatalf("ParsePrivateKey(hex) failed: %v", err)
	}
	if hexKey != testHexPriv {
		t.Fatalf("expected %s, got %s", testHexPriv, hexKey)
	}

	// Test EncodeNsec
	nsec, err := EncodeNsec(testHexPriv)
	if err != nil {
		t.Fatalf("EncodeNsec failed: %v", err)
	}
	if nsec == "" {
		t.Fatalf("expected non-empty nsec")
	}

	// Test ParsePrivateKey with nsec
	parsedFromNsec, err := ParsePrivateKey(nsec)
	if err != nil {
		t.Fatalf("ParsePrivateKey(nsec) failed: %v", err)
	}
	if parsedFromNsec != testHexPriv {
		t.Fatalf("expected %s, got %s", testHexPriv, parsedFromNsec)
	}

	// Test GetPublicKeyHex
	pubHex, err := GetPublicKeyHex(nsec)
	if err != nil {
		t.Fatalf("GetPublicKeyHex failed: %v", err)
	}
	if len(pubHex) != 64 {
		t.Fatalf("expected 64 hex chars for pubkey, got %d (%s)", len(pubHex), pubHex)
	}

	// Test EncodeNpub
	npub, err := EncodeNpub(pubHex)
	if err != nil {
		t.Fatalf("EncodeNpub failed: %v", err)
	}
	if npub == "" {
		t.Fatalf("expected non-empty npub")
	}

	// Test MaskNsec
	masked := MaskNsec(nsec)
	if masked == nsec || len(masked) == 0 {
		t.Fatalf("masking did not mask properly: %s", masked)
	}
}
