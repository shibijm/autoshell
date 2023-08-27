package crypto

import (
	"crypto/rand"
	"errors"
	"io"
	"runtime"

	"golang.org/x/crypto/argon2"
)

var kdfSaltLength = 32

func deriveKey(password string, salt []byte, keyLengthBytes uint32) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, kdfSaltLength)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, nil, err
		}
	} else if len(salt) != kdfSaltLength {
		return nil, nil, errors.New("invalid salt length")
	}
	return argon2.IDKey([]byte(password), salt, 8, 16*1024, uint8(runtime.NumCPU()), keyLengthBytes), salt, nil
}
