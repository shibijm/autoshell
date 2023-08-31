package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"autoshell/utils"
	"bytes"
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type configService struct {
	crypter         ports.Crypter
	filePath        string
	encryptionMark  []byte
	idLength        int
	isFileEncrypted bool
	configBytes     []byte
	config          *entities.Config
}

func NewConfigService(crypter ports.Crypter, filePath string, getPassword ports.PasswordFactory) (ports.ConfigService, error) {
	encryptionMark := []byte{0x17, 0x6F, 0x95, 0xF3, 0xF3, 0x81, 0x32, 0x6F}
	encryptionMarkLength := len(encryptionMark)
	idLength := 32
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	isFileEncrypted := bytes.Equal(fileData[:encryptionMarkLength], encryptionMark)
	var configBytes []byte
	if isFileEncrypted {
		if len(fileData) < encryptionMarkLength+idLength {
			return nil, errors.New("invalid encrypted config file")
		}
		id := fileData[encryptionMarkLength : encryptionMarkLength+idLength]
		password, err := getPassword(id)
		if err != nil {
			return nil, err
		}
		configBytes, err = crypter.Decrypt(fileData[encryptionMarkLength+idLength:], password)
		if err != nil {
			return nil, err
		}
	} else {
		configBytes = fileData
	}
	config := entities.Config{}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}
	return &configService{crypter, filePath, encryptionMark, idLength, isFileEncrypted, configBytes, &config}, nil
}

func (s *configService) GetConfig() *entities.Config {
	return s.config
}

func (s *configService) SaveToFileEncrypted(getPassword ports.PasswordFactory) error {
	if s.isFileEncrypted {
		return errors.New("config file is already encrypted")
	}
	id := utils.GenerateRandomBytes(s.idLength)
	password, err := getPassword(id)
	if err != nil {
		return err
	}
	data, err := s.crypter.Encrypt(s.configBytes, password)
	if err != nil {
		return err
	}
	err = os.WriteFile(s.filePath, append(append(s.encryptionMark, id...), data...), 0600)
	if err != nil {
		return err
	}
	s.isFileEncrypted = true
	return nil
}

func (s *configService) SaveToFileDecrypted() error {
	if !s.isFileEncrypted {
		return errors.New("config file is not encrypted")
	}
	err := os.WriteFile(s.filePath, s.configBytes, 0600)
	if err != nil {
		return err
	}
	s.isFileEncrypted = false
	return nil
}
