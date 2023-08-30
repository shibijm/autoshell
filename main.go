package main

import (
	"autoshell/cli"
	"autoshell/core/ports"
	"autoshell/core/services"
	"autoshell/crypto"
	"os"
)

const version = "1.0.0"

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
		os.Exit(1)
	}
}
