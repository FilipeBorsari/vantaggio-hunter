package crypto_test

import (
	"bytes"
	"testing"

	"github.com/vantaggio/prospect-api/pkg/crypto"
)

var testKey = bytes.Repeat([]byte("k"), 32)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("minha-api-key-secreta")

	enc, err := crypto.Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == "" {
		t.Fatal("encrypted string is empty")
	}

	dec, err := crypto.Decrypt(enc, testKey)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(dec, plaintext) {
		t.Errorf("round-trip mismatch: got %q, want %q", dec, plaintext)
	}
}

func TestEncryptProducesUniqueOutputs(t *testing.T) {
	plaintext := []byte("mesmo-input")

	enc1, err := crypto.Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := crypto.Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}
	// Different nonces → different ciphertexts.
	if enc1 == enc2 {
		t.Error("expected different ciphertexts for same plaintext (different nonces)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	enc, err := crypto.Encrypt([]byte("secret"), testKey)
	if err != nil {
		t.Fatal(err)
	}

	wrongKey := bytes.Repeat([]byte("x"), 32)
	if _, err := crypto.Decrypt(enc, wrongKey); err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	if _, err := crypto.Decrypt("!!!not-base64!!!", testKey); err == nil {
		t.Error("expected error for invalid base64 input")
	}
}

func TestDecryptTooShort(t *testing.T) {
	// Valid base64 but too short to contain a nonce.
	import64 := "YQ==" // base64("a") — 1 byte, less than AES-GCM nonce size
	if _, err := crypto.Decrypt(import64, testKey); err == nil {
		t.Error("expected error for ciphertext shorter than nonce")
	}
}
