package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/denisbrodbeck/machineid"
	"golang.org/x/crypto/argon2"
)

var devicePassSeed = "hSVpIyPHZCvfbyPZ4qVqFbLWT9783kkNvw9Pd9hnKBCERCuphJzHjeVYMnnag9MWag3SJxQL2HwCSCKyYpWvf8syLRMpzVGRE2USgPsSrFzHvfvpwACr88aDgzyQuWsZ"

func GenerateDevicePass(salt []byte) (string, error) {
	machineID, err := machineid.ID()
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write([]byte(machineID))
	hash.Write([]byte(devicePassSeed))
	hash.Write(salt)
	hashBytes := hash.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)
	return hashHex, nil
}

func GenerateRandomBytes(length int) []byte {
	randomBytes := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		panic(err)
	}
	return randomBytes
}

func GenerateArgon2idKey(password string, salt []byte, length int) []byte {
	return argon2.IDKey([]byte(password), salt, 8, 16*1024, 8, uint32(length))
}
