package cli

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"autoshell/utils"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type configServiceFactory func(filePath string, getPassword ports.PasswordFactory) (ports.ConfigService, error)

type runnerFactory func(config *entities.Config) ports.Runner

type cliController struct {
	version             string
	createConfigService configServiceFactory
	createRunner        runnerFactory
	devicePassVar       string
}

func NewCliController(version string, createConfigService configServiceFactory, createRunner runnerFactory) *cliController {
	return &cliController{version, createConfigService, createRunner, "$auto"}
}

func (c *cliController) Execute() error {
	var configPath string
	rootCmd := &cobra.Command{
		Use:                "autoshell",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableSuggestions: true,
		Version:            c.version,
	}
	runCmd := &cobra.Command{
		Use:   "run [workflow] [args]",
		Short: "Run a workflow",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configService, err := c.createConfigService(configPath, utils.GenerateDevicePass)
			if err != nil {
				configService, err = c.createConfigService(configPath, func(id []byte) (string, error) {
					return c.readPassword(id, false)
				})
				if err != nil {
					return err
				}
			}
			config := configService.GetConfig()
			runner := c.createRunner(config)
			return runner.Run(args[0], args[1:])
		},
	}
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config file management",
	}
	configEncryptCmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt the config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configService, err := c.createConfigService(configPath, func(id []byte) (string, error) {
				return "", errors.New("config file is already encrypted")
			})
			if err != nil {
				return err
			}
			var password string
			var devicePassVarUsed bool
			err = configService.SaveToFileEncrypted(
				func(id []byte) (string, error) {
					password, err, devicePassVarUsed = c.readPasswordAndDpvu(id, true)
					return password, err
				},
			)
			if err != nil {
				return err
			}
			if devicePassVarUsed {
				fmt.Printf("Password contains \"%s\"\n", c.devicePassVar)
				if configService.GetConfig().Protected {
					fmt.Printf("Config file is marked as protected and hence cannot be saved after decryption if the decryption password contains \"%s\"\n", c.devicePassVar)
					fmt.Println("Please store this explicit password safely: " + password)
				}
			}
			fmt.Println("Config file encrypted successfully")
			return nil
		},
	}
	configDecryptCmd := &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt the config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var devicePassVarUsed bool
			configService, err := c.createConfigService(configPath, func(id []byte) (string, error) {
				var password string
				var err error
				password, err, devicePassVarUsed = c.readPasswordAndDpvu(id, false)
				return password, err
			})
			if err != nil {
				return err
			}
			if configService.GetConfig().Protected && devicePassVarUsed {
				return fmt.Errorf("config file is marked as protected, refusing to save the decrypted data to disk since the decryption password contains \"%s\"", c.devicePassVar)
			}
			err = configService.SaveToFileDecrypted()
			if err != nil {
				return err
			}
			fmt.Println("Config file decrypted successfully")
			return nil
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yml", "config file path")
	rootCmd.AddCommand(runCmd, configCmd)
	configCmd.AddCommand(configEncryptCmd, configDecryptCmd)
	return rootCmd.Execute()
}

func (c *cliController) readPasswordAndDpvu(id []byte, confirm bool) (string, error, bool) {
	password, err := utils.ReadInputHidden("Password")
	if err != nil {
		return "", err, false
	}
	if confirm {
		confirmationPassword, err := utils.ReadInputHidden("Confirm Password")
		if err == nil && confirmationPassword != password {
			err = errors.New("the two passwords didn't match")
		}
		if err != nil {
			return "", err, false
		}
	}
	if !strings.Contains(password, c.devicePassVar) {
		return password, nil, false
	}
	devicePass, err := utils.GenerateDevicePass(id)
	if err != nil {
		return "", err, false
	}
	password = strings.ReplaceAll(password, c.devicePassVar, devicePass)
	return password, nil, true
}

func (c *cliController) readPassword(id []byte, confirm bool) (string, error) {
	password, err, _ := c.readPasswordAndDpvu(id, confirm)
	return password, err
}
