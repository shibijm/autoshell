package crypto

import (
	"autoshell/core/ports"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

type aesGcmCrypter struct {
	nonceLength int
}

func NewAesGcmCrypter() ports.Crypter {
	return &aesGcmCrypter{12}
}

func (c *aesGcmCrypter) Encrypt(data []byte, password string) ([]byte, error) {
	key, kdfSalt, err := deriveKey(password, nil, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, c.nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	encryptedData := gcm.Seal(nil, nonce, data, nil)
	payload := append(kdfSalt, append(nonce, encryptedData...)...)
	return payload, nil
}

func (c *aesGcmCrypter) Decrypt(payload []byte, password string) ([]byte, error) {
	encryptedDataLength := len(payload) - (kdfSaltLength + c.nonceLength)
	if encryptedDataLength < 0 {
		return nil, errors.New("invalid payload")
	}
	kdfSalt := payload[:kdfSaltLength]
	key, _, err := deriveKey(password, kdfSalt, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := payload[kdfSaltLength : kdfSaltLength+c.nonceLength]
	encryptedData := payload[kdfSaltLength+c.nonceLength:]
	data, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}
