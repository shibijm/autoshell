package ports

import "autoshell/core/entities"

type ConfigService interface {
	ParseConfigFile(filePath string) (*entities.Config, error)
	ParseConfigFileWithMachineID(filePath string) (*entities.Config, error)
	ParseConfigFileWithPassword(filePath string, password string) (*entities.Config, error)
	IsEncryptedConfigFile(filePath string) (bool, error)
	DecryptConfigFile(filePath string, password string) error
	EncryptConfigFile(filePath string, password string) error
	ContainsMachineID(password string) bool
}

type Runner interface {
	Run(config *entities.Config, workflowName string) error
}
