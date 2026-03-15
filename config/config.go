package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const devicePassVar = "$DP"

var (
	magicBytes        = []byte{0x17, 0x6F, 0x95, 0xF3, 0xF3, 0x81, 0x32, 0x6F}
	magicBytesLen     = len(magicBytes)
	devicePassSaltLen = 32
)

type Config struct {
	Protected bool              `yaml:"protected"`
	Workflows map[string]string `yaml:"workflows"`
}

type GetPassword func() (string, error)

type loadResult struct {
	filePath          string
	isFileEncrypted   bool
	autoDecrypted     bool
	devicePassVarUsed bool
	configBytes       []byte
	config            Config
}

func load(filePath string, attemptAutoDecrypt bool, getPassword GetPassword) (*loadResult, error) {
	filePaths := []string{filePath}
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}
	filePaths = append(filePaths, filepath.Join(filepath.Dir(exe), filePath))
	if runtime.GOOS == "linux" {
		filePaths = append(filePaths, filepath.Join("/etc/autoshell", filePath))
	}
	var fileData []byte
	for _, filePath = range filePaths {
		fileData, err = os.ReadFile(filePath) //nolint:gosec
		if err == nil {
			break
		}
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return nil, fmt.Errorf("read file: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("non-existent files: %s", strings.Join(filePaths, ", "))
	}
	if len(fileData) < magicBytesLen {
		return nil, errors.New("file too short")
	}
	isFileEncrypted := bytes.Equal(fileData[:magicBytesLen], magicBytes)
	var autoDecrypted bool
	var devicePassVarUsed bool
	var configBytes []byte
	if !isFileEncrypted {
		configBytes = fileData
	} else if attemptAutoDecrypt || getPassword != nil {
		if len(fileData) < magicBytesLen+devicePassSaltLen {
			return nil, errors.New("invalid encrypted file")
		}
		devicePassSalt := fileData[magicBytesLen : magicBytesLen+devicePassSaltLen]
		devicePass := generateDevicePass(devicePassSalt)
		if attemptAutoDecrypt {
			if data, err := aesGcmDecrypt(fileData[magicBytesLen+devicePassSaltLen:], devicePass); err == nil {
				autoDecrypted = true
				configBytes = data
			}
		}
		if configBytes == nil && getPassword != nil {
			var password string
			password, devicePassVarUsed, err = readPassword(getPassword, devicePass)
			if err != nil {
				return nil, fmt.Errorf("read password: %w", err)
			}
			configBytes, err = aesGcmDecrypt(fileData[magicBytesLen+devicePassSaltLen:], password)
			if err != nil {
				return nil, fmt.Errorf("decrypt: %w", err)
			}
		}
	}
	var config Config
	if configBytes != nil {
		if err := yaml.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
	}
	return &loadResult{
		filePath:          filePath,
		isFileEncrypted:   isFileEncrypted,
		autoDecrypted:     autoDecrypted,
		devicePassVarUsed: devicePassVarUsed,
		configBytes:       configBytes,
		config:            config,
	}, nil
}

func Get(filePath string, getPassword GetPassword) (Config, error) {
	r, err := load(filePath, true, getPassword)
	if err != nil {
		return Config{}, fmt.Errorf("load: %w", err)
	}
	return r.config, nil
}

func Encrypt(filePath string, getPassword GetPassword) (string, error) {
	r, err := load(filePath, false, nil)
	if err != nil {
		return "", fmt.Errorf("load: %w", err)
	}
	if r.isFileEncrypted {
		return "", errors.New("already encrypted")
	}
	devicePassSalt := generateRandomBytes(devicePassSaltLen)
	devicePass := generateDevicePass(devicePassSalt)
	password, devicePassVarUsed, err := readPassword(getPassword, devicePass)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	payload, err := aesGcmEncrypt(r.configBytes, password)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if err := atomicWrite(r.filePath, func(file *os.File) error {
		_, err := file.Write(slices.Concat(magicBytes, devicePassSalt, payload))
		return err
	}); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	message := new(strings.Builder)
	if devicePassVarUsed {
		fmt.Fprintf(message, "%s = %s", devicePassVar, devicePass)
		if r.config.Protected {
			fmt.Fprintf(message, "\nConfig file is marked as protected and hence cannot be saved decrypted without substituting %q", devicePassVar)
		}
	}
	return message.String(), nil
}

func Decrypt(filePath string, getPassword GetPassword) error {
	r, err := load(filePath, true, getPassword)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if !r.isFileEncrypted {
		return errors.New("already decrypted")
	}
	if r.config.Protected {
		if r.autoDecrypted {
			r, err = load(filePath, false, getPassword)
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}
		}
		if r.devicePassVarUsed {
			return fmt.Errorf("file is protected and the password contains %q", devicePassVar)
		}
	}
	if err := atomicWrite(r.filePath, func(file *os.File) error {
		_, err := file.Write(r.configBytes)
		return err
	}); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func readPassword(getPassword GetPassword, devicePass string) (string, bool, error) {
	password, err := getPassword()
	if err != nil {
		return "", false, err
	}
	var devicePassVarUsed bool
	if strings.Contains(password, devicePassVar) {
		password = strings.ReplaceAll(password, devicePassVar, devicePass)
		devicePassVarUsed = true
	}
	return password, devicePassVarUsed, nil
}

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

func atomicWrite(filePath string, write func(file *os.File) error) error {
	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, dirPerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmpFile, err := os.CreateTemp(dirPath, filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	tmpFilePath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpFilePath)
		}
	}()
	if err := tmpFile.Chmod(filePerm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod: %w", err)
	}
	if err := write(tmpFile); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpFilePath, filePath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	cleanup = false
	return nil
}
