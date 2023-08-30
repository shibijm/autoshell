package crypto

import (
	"autoshell/core/ports"
	"autoshell/utils"
	"crypto/aes"
	"crypto/cipher"
	"errors"

	"golang.org/x/crypto/argon2"
)

type aesGcmCrypter struct {
	saltLength  int
	nonceLength int
}

func NewAesGcmCrypter() ports.Crypter {
	return &aesGcmCrypter{32, 12}
}

func (c *aesGcmCrypter) Encrypt(data []byte, password string) ([]byte, error) {
	salt := utils.GenerateRandomBytes(c.saltLength)
	key := c.generateArgon2idKey(password, salt)
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
	encryptedDataLength := len(payload) - (c.saltLength + c.nonceLength)
	if encryptedDataLength < 0 {
		return nil, errors.New("invalid payload")
	}
	salt := payload[:c.saltLength]
	key := c.generateArgon2idKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := payload[c.saltLength : c.saltLength+c.nonceLength]
	encryptedData := payload[c.saltLength+c.nonceLength:]
	data, err := gcm.Open(nil, nonce, encryptedData, nil)
	return data, err
}

func (c *aesGcmCrypter) generateArgon2idKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 8, 16*1024, 8, 32)
}
