package ports

import "autoshell/core/entities"

type Crypter interface {
	Encrypt(data []byte, password string) ([]byte, error)
	Decrypt(payload []byte, password string) ([]byte, error)
}

type PasswordFactory func(id []byte) (string, error)

type ConfigService interface {
	GetConfig() *entities.Config
	SaveToFileEncrypted(getPassword PasswordFactory) error
	SaveToFileDecrypted() error
}

type Runner interface {
	Run(workflowName string, args []string) error
}
