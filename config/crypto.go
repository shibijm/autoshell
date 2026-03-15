package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/denisbrodbeck/machineid"
	"golang.org/x/crypto/argon2"
)

const (
	keyLength   = 32
	saltLength  = 32
	nonceLength = 12
)

func aesGcmEncrypt(data []byte, password string) ([]byte, error) {
	salt := generateRandomBytes(saltLength)
	key := generateArgon2IdKey(password, salt, keyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new GCM: %w", err)
	}
	nonce := generateRandomBytes(nonceLength)
	encryptedData := gcm.Seal(nil, nonce, data, nil)
	payload := slices.Concat(salt, nonce, encryptedData)
	return payload, nil
}

func aesGcmDecrypt(payload []byte, password string) ([]byte, error) {
	if len(payload) < saltLength+nonceLength {
		return nil, errors.New("invalid payload")
	}
	salt := payload[:saltLength]
	key := generateArgon2IdKey(password, salt, keyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new GCM: %w", err)
	}
	nonce := payload[saltLength : saltLength+nonceLength]
	encryptedData := payload[saltLength+nonceLength:]
	data, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("GCM open: %w", err)
	}
	return data, nil
}

func generateArgon2IdKey(password string, salt []byte, keyLength uint32) []byte {
	return argon2.IDKey([]byte(password), salt, 8, 16*1024, 8, keyLength)
}

func generateRandomBytes(length int) []byte {
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Errorf("read random bytes: %w", err))
	}
	return randomBytes
}

var devicePassSeed = "hSVpIyPHZCvfbyPZ4qVqFbLWT9783kkNvw9Pd9hnKBCERCuphJzHjeVYMnnag9MWag3SJxQL2HwCSCKyYpWvf8syLRMpzVGRE2USgPsSrFzHvfvpwACr88aDgzyQuWsZ" //nolint:gosec

var machineId = func() string {
	id, err := machineid.ID()
	if err != nil {
		panic(err)
	}
	return id
}()

func generateDevicePass(salt []byte) string {
	hash := sha256.New()
	hash.Write([]byte(machineId))
	hash.Write([]byte(devicePassSeed))
	hash.Write(salt)
	hashBytes := hash.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)
	return hashHex
}
