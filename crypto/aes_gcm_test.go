package crypto

import (
	"bytes"
	"testing"
)

func TestAesGcmCrypter(t *testing.T) {
	crypter := NewAesGcmCrypter()
	data := []byte("Test data")
	password := "testPassword"
	t.Run("Encrypt and Decrypt", func(t *testing.T) {
		encryptedData, err := crypter.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encrypt failed: %s", err)
		}
		decryptedData, err := crypter.Decrypt(encryptedData, password)
		if err != nil {
			t.Fatalf("Decrypt failed: %s", err)
		}
		if !bytes.Equal(data, decryptedData) {
			t.Fatal("Decrypted data does not match original data")
		}
	})
	t.Run("Invalid Password", func(t *testing.T) {
		encryptedData, err := crypter.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encrypt failed: %v", err)
		}
		_, err = crypter.Decrypt(encryptedData, "wrongPassword")
		if err == nil {
			t.Fatalf("Decrypt should have failed with invalid password")
		}
	})
	t.Run("Invalid Payload", func(t *testing.T) {
		encryptedData := []byte{4, 8, 15, 16, 23, 42}
		_, err := crypter.Decrypt(encryptedData, password)
		if err == nil {
			t.Fatalf("Decrypt should have failed with invalid payload")
		}
	})
	t.Run("Modified Payload", func(t *testing.T) {
		encryptedData, err := crypter.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encrypt failed: %s", err)
		}
		encryptedData[len(encryptedData)/2]++
		_, err = crypter.Decrypt(encryptedData, password)
		if err == nil {
			t.Fatalf("Decrypt should have failed with modified payload")
		}
	})
	t.Run("Non-Deterministic Encryption", func(t *testing.T) {
		encryptedData1, err := crypter.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encrypt failed: %s", err)
		}
		encryptedData2, err := crypter.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encrypt failed: %s", err)
		}
		if bytes.Equal(encryptedData1, encryptedData2) {
			t.Fatal("Encrypted outputs are unexpectedly the same")
		}
	})
}
