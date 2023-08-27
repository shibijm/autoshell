package crypto

import (
	"autoshell/core/ports"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

type aesCbcCrypter struct{}

func NewAesCbcCrypter() ports.Crypter {
	return &aesCbcCrypter{}
}

func (c *aesCbcCrypter) Encrypt(data []byte, password string) ([]byte, error) {
	key, kdfSalt, err := deriveKey(password, nil, 64)
	if err != nil {
		return nil, err
	}
	aesKey := key[:32]
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCEncrypter(block, iv)
	data = pad(data, aes.BlockSize)
	encryptedData := make([]byte, len(data))
	cbc.CryptBlocks(encryptedData, data)
	payload := append(kdfSalt, append(iv, encryptedData...)...)
	macKey := key[32:]
	hash := hmac.New(sha256.New, macKey)
	hash.Write(payload)
	hashBytes := hash.Sum(nil)
	payload = append(payload, hashBytes...)
	return payload, nil
}

func (c *aesCbcCrypter) Decrypt(payload []byte, password string) ([]byte, error) {
	payloadLength := len(payload)
	encryptedDataLength := payloadLength - (kdfSaltLength + aes.BlockSize + sha256.Size)
	if encryptedDataLength < 0 || encryptedDataLength%aes.BlockSize != 0 {
		return nil, errors.New("invalid payload")
	}
	kdfSalt := payload[:kdfSaltLength]
	key, _, err := deriveKey(password, kdfSalt, 64)
	if err != nil {
		return nil, err
	}
	macKey := key[32:]
	hash := hmac.New(sha256.New, macKey)
	hash.Write(payload[:payloadLength-sha256.Size])
	calculatedHashBytes := hash.Sum(nil)
	hashBytes := payload[payloadLength-sha256.Size:]
	if !hmac.Equal(calculatedHashBytes, hashBytes) {
		return nil, errors.New("HMAC validation failed")
	}
	aesKey := key[:32]
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	iv := payload[kdfSaltLength : kdfSaltLength+aes.BlockSize]
	cbc := cipher.NewCBCDecrypter(block, iv)
	encryptedData := payload[kdfSaltLength+aes.BlockSize : payloadLength-sha256.Size]
	data := make([]byte, len(encryptedData))
	cbc.CryptBlocks(data, encryptedData)
	data = unpad(data)
	return data, nil
}

func pad(source []byte, blockSize int) []byte {
	paddingLength := blockSize - len(source)%blockSize
	padding := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	return append(source, padding...)
}

func unpad(source []byte) []byte {
	length := len(source)
	if length == 0 {
		return source
	}
	paddingLength := int(source[length-1])
	if length < paddingLength {
		return source
	}
	return source[:(length - paddingLength)]
}
