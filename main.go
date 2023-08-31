package main

import (
	"autoshell/cli"
	"autoshell/core/ports"
	"autoshell/core/services"
	"autoshell/crypto"
	"fmt"
	"os"
	"unicode"
)

const version = "1.0.1"

func main() {
	crypter := crypto.NewAesGcmCrypter()
	cliController := cli.NewCliController(
		version,
		func(filePath string, getPassword ports.PasswordFactory) (ports.ConfigService, error) {
			return services.NewConfigService(crypter, filePath, getPassword)
		},
		services.NewRunner,
	)
	err := cliController.Execute()
	if err != nil {
		errRunes := []rune(err.Error())
		errRunes[0] = unicode.ToUpper(errRunes[0])
		errString := string(errRunes)
		fmt.Printf("Error: %s\n", errString)
		os.Exit(1)
	}
}
