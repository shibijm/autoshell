package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type configService struct {
	crypter              ports.Crypter
	encryptedConfigMark  []byte
	machineID            string
	machineIDPlaceholder string
}

func NewConfigService(crypter ports.Crypter, encryptedConfigMark []byte, machineID string, machineIDPlaceholder string) ports.ConfigService {
	return &configService{crypter, encryptedConfigMark, machineID, machineIDPlaceholder}
}

func (s *configService) parseConfigFile(filePath string, password string) (*entities.Config, error) {
	payload, err := os.ReadFile(filePath)
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

func (s *configService) ParseConfigFile(filePath string) (*entities.Config, error) {
	return s.parseConfigFile(filePath, "")
}

func (s *configService) ParseConfigFileWithMachineID(filePath string) (*entities.Config, error) {
	return s.parseConfigFile(filePath, s.machineIDPlaceholder)
}

func (s *configService) ParseConfigFileWithPassword(filePath string, password string) (*entities.Config, error) {
	return s.parseConfigFile(filePath, password)
}

func (s *configService) IsEncryptedConfigFile(filePath string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	b := make([]byte, 8)
	_, err = f.Read(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(b, s.encryptedConfigMark), nil
}

func (s *configService) EncryptConfigFile(filePath string, password string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	encryptedData, err := s.crypter.Encrypt(data, s.transformPassword(password))
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, append(s.encryptedConfigMark, encryptedData...), 0700)
	return err
}

func (s *configService) DecryptConfigFile(filePath string, password string) error {
	payload, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	data, err := s.crypter.Decrypt(payload[len(s.encryptedConfigMark):], s.transformPassword(password))
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0700)
	return err
}

func (s *configService) ContainsMachineID(password string) bool {
	return strings.Contains(password, s.machineIDPlaceholder)
}

func (s *configService) transformPassword(password string) string {
	return strings.ReplaceAll(password, s.machineIDPlaceholder, s.machineID)
}
