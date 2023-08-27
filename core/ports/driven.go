package ports

type Crypter interface {
	Encrypt(data []byte, password string) ([]byte, error)
	Decrypt(payload []byte, password string) ([]byte, error)
}
