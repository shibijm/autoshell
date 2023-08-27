package crypto

import (
	"autoshell/core/ports"
	"bytes"
	"testing"
)

func TestAesCbcCrypter(t *testing.T) {
	testAesCrypter(t, NewAesCbcCrypter())
}

func TestAesGcmCrypter(t *testing.T) {
	testAesCrypter(t, NewAesGcmCrypter())
}

func testAesCrypter(t *testing.T, crypter ports.Crypter) {
	data := []byte("Hello world!")
	password := "random password"
	encryptedData, err := crypter.Encrypt(data, password)
	if err != nil {
		t.Fatalf("failed to encrypt: %s", err)
	}
	decryptedData, err := crypter.Decrypt(encryptedData, password)
	if err != nil {
		t.Fatalf("failed to decrypt: %s", err)
	}
	if !bytes.Equal(data, decryptedData) {
		t.Fatal("decrypted data does not match the original data")
	}
}
