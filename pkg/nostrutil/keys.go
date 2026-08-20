package nostrutil

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// ParsePrivateKey parses a hex or bech32 nsec private key string and returns the hex private key.
func ParsePrivateKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("empty private key")
	}

	if strings.HasPrefix(key, "nsec1") {
		prefix, val, err := nip19.Decode(key)
		if err != nil {
			return "", fmt.Errorf("failed to decode nsec: %w", err)
		}
		if prefix != "nsec" {
			return "", fmt.Errorf("expected nsec prefix, got %s", prefix)
		}
		hexKey, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("unexpected decoded nsec type")
		}
		return hexKey, nil
	}

	// Validate hex
	b, err := hex.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("invalid hex private key: %w", err)
	}
	if len(b) != 32 {
		return "", fmt.Errorf("private key must be 32 bytes, got %d", len(b))
	}

	return key, nil
}

// GetPublicKeyHex derives the 32-byte schnorr public key (hex) from a private key (hex or nsec).
func GetPublicKeyHex(privKey string) (string, error) {
	hexKey, err := ParsePrivateKey(privKey)
	if err != nil {
		return "", err
	}

	b, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", err
	}

	_, pub := btcec.PrivKeyFromBytes(b)
	pubBytes := pub.SerializeCompressed()[1:] // 32-byte x-only pubkey for schnorr
	return hex.EncodeToString(pubBytes), nil
}

// EncodeNpub converts a hex public key to bech32 npub.
func EncodeNpub(pubHex string) (string, error) {
	pubHex = strings.TrimSpace(pubHex)
	return nip19.EncodePublicKey(pubHex)
}

// EncodeNsec converts a hex private key to bech32 nsec.
func EncodeNsec(privHex string) (string, error) {
	privHex = strings.TrimSpace(privHex)
	return nip19.EncodePrivateKey(privHex)
}

// MaskNsec safely returns a masked representation of an nsec or hex key.
func MaskNsec(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 10 {
		return "******"
	}
	return key[:6] + "..." + key[len(key)-4:]
}
