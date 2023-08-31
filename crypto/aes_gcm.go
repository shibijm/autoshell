package crypto

import (
	"autoshell/core/ports"
	"autoshell/utils"
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

type aesGcmCrypter struct {
	keyLength    int
	pwSaltLength int
	nonceLength  int
}

func NewAesGcmCrypter() ports.Crypter {
	return &aesGcmCrypter{32, 32, 12}
}

func (c *aesGcmCrypter) Encrypt(data []byte, password string) ([]byte, error) {
	salt := utils.GenerateRandomBytes(c.pwSaltLength)
	key := utils.GenerateArgon2idKey(password, salt, c.keyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := utils.GenerateRandomBytes(c.nonceLength)
	encryptedData := gcm.Seal(nil, nonce, data, nil)
	payload := append(append(salt, nonce...), encryptedData...)
	return payload, nil
}

func (c *aesGcmCrypter) Decrypt(payload []byte, password string) ([]byte, error) {
	if len(payload) < c.pwSaltLength+c.nonceLength {
		return nil, errors.New("invalid payload")
	}
	salt := payload[:c.pwSaltLength]
	key := utils.GenerateArgon2idKey(password, salt, c.keyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := payload[c.pwSaltLength : c.pwSaltLength+c.nonceLength]
	encryptedData := payload[c.pwSaltLength+c.nonceLength:]
	return gcm.Open(nil, nonce, encryptedData, nil)
}
