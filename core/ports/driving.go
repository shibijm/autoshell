package ports

import "autoshell/core/entities"

type ConfigFileService interface {
	ParseFile() (*entities.Config, error)
	ParseFileWithMachineID() (*entities.Config, error)
	ParseFileWithPassword(password string) (*entities.Config, error)
	IsFileEncrypted() (bool, error)
	DecryptFile(password string) error
	EncryptFile(password string) error
	DoesContainMachineID(password string) bool
}

type Runner interface {
	Run(workflowName string, args []string) error
}
