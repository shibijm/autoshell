package cli

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"autoshell/utils"
	"errors"
	"fmt"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
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
					password, _, err := c.readPassword("Password", id)
					return password, err
				})
				if err != nil {
					return err
				}
			}
			runner := c.createRunner(configService.GetConfig())
			err = runner.Run(args[0], args[1:])
			return err
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
					password, devicePassVarUsed, err = c.readPassword("Password", id)
					return password, err
				},
				func(id []byte) (string, error) {
					password, _, err = c.readPassword("Confirm Password", id)
					return password, err
				},
			)
			if err != nil {
				return err
			}
			if devicePassVarUsed {
				fmt.Println("Password contains device pass variable")
				if configService.GetConfig().Protected {
					fmt.Println("Config file is marked as protected and hence cannot be saved decrypted if opened with a password containing device pass variable")
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
				password, devicePassVarUsed, err = c.readPassword("Password", id)
				return password, err
			})
			if err != nil {
				return err
			}
			if configService.GetConfig().Protected && devicePassVarUsed {
				return errors.New("config file is marked as protected, refusing to save decrypted as it was opened with a password containing device pass variable")
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

func (c *cliController) readPassword(prompt string, id []byte) (string, bool, error) {
	fmt.Print(prompt + ": ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err == nil && len(passwordBytes) == 0 {
		err = errors.New("password is empty")
	}
	if err != nil {
		return "", false, err
	}
	password := string(passwordBytes)
	if !strings.Contains(password, c.devicePassVar) {
		return password, false, nil
	}
	devicePass, err := utils.GenerateDevicePass(id)
	if err != nil {
		return "", false, err
	}
	password = strings.ReplaceAll(password, c.devicePassVar, devicePass)
	return password, true, nil
}
