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
		t.Fatal(err)
	}
	decryptedData, err := crypter.Decrypt(encryptedData, password)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decryptedData) {
		t.Fatal("wrong decrypted data")
	}
}
