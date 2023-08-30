package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/denisbrodbeck/machineid"
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
