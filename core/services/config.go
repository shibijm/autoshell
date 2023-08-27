package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type configFileService struct {
	filePath             string
	crypter              ports.Crypter
	encryptedConfigMark  []byte
	machineID            string
	machineIDPlaceholder string
}

func NewConfigFileService(filePath string, crypter ports.Crypter, encryptedConfigMark []byte, machineID string, machineIDPlaceholder string) ports.ConfigFileService {
	return &configFileService{filePath, crypter, encryptedConfigMark, machineID, machineIDPlaceholder}
}

func (s *configFileService) parseConfigFile(password string) (*entities.Config, error) {
	payload, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}
	var data []byte
	if password != "" {
		data, err = s.crypter.Decrypt(payload[len(s.encryptedConfigMark):], s.transformPassword(password))
		if err != nil {
			return nil, err
		}
	} else {
		data = payload
	}
	config := entities.Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *configFileService) ParseFile() (*entities.Config, error) {
	return s.parseConfigFile("")
}

func (s *configFileService) ParseFileWithMachineID() (*entities.Config, error) {
	return s.parseConfigFile(s.machineIDPlaceholder)
}

func (s *configFileService) ParseFileWithPassword(password string) (*entities.Config, error) {
	return s.parseConfigFile(password)
}

func (s *configFileService) IsFileEncrypted() (bool, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	b := make([]byte, len(s.encryptedConfigMark))
	_, err = file.Read(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(b, s.encryptedConfigMark), nil
}

func (s *configFileService) EncryptFile(password string) error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	encryptedData, err := s.crypter.Encrypt(data, s.transformPassword(password))
	if err != nil {
		return err
	}
	err = os.WriteFile(s.filePath, append(s.encryptedConfigMark, encryptedData...), 0600)
	return err
}

func (s *configFileService) DecryptFile(password string) error {
	payload, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	data, err := s.crypter.Decrypt(payload[len(s.encryptedConfigMark):], s.transformPassword(password))
	if err != nil {
		return err
	}
	err = os.WriteFile(s.filePath, data, 0700)
	return err
}

func (s *configFileService) DoesContainMachineID(password string) bool {
	return strings.Contains(password, s.machineIDPlaceholder)
}

func (s *configFileService) transformPassword(password string) string {
	return strings.ReplaceAll(password, s.machineIDPlaceholder, s.machineID)
}
