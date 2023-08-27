package cli

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"autoshell/utils"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type configFileServiceFactory func(filePath string) ports.ConfigFileService

type runnerFactory func(config *entities.Config) ports.Runner

type cliController struct {
	version      string
	createCfs    configFileServiceFactory
	createRunner runnerFactory
}

func NewCliController(version string, createCfs configFileServiceFactory, createRunner runnerFactory) *cliController {
	return &cliController{version, createCfs, createRunner}
}

func (cli *cliController) Execute() error {
	var configPath string
	var cfs ports.ConfigFileService
	cobra.OnInitialize(func() {
		_, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			utils.ExitWithError(fmt.Errorf("config file '%s' does not exist", configPath))
		}
		cfs = cli.createCfs(configPath)
	})
	rootCmd := &cobra.Command{
		Use:                "autoshell",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableSuggestions: true,
		Version:            cli.version,
	}
	runCmd := &cobra.Command{
		Use:   "run workflow",
		Short: "Run a workflow",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isEncrypted, err := cfs.IsFileEncrypted()
			if err != nil {
				return err
			}
			var config *entities.Config
			if isEncrypted {
				config, err = cfs.ParseFileWithMachineID()
				if err != nil {
					password, pwdErr := cli.readPassword("Password")
					if pwdErr != nil {
						return pwdErr
					}
					config, err = cfs.ParseFileWithPassword(password)
				}
			} else {
				config, err = cfs.ParseFile()
			}
			if err != nil {
				return err
			}
			runner := cli.createRunner(config)
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
			isEncrypted, err := cfs.IsFileEncrypted()
			if err != nil {
				return err
			}
			if isEncrypted {
				return errors.New("config file is already encrypted")
			}
			password, err := cli.readPassword("Password")
			if err != nil {
				return err
			}
			confirmPassword, err := cli.readPassword("Confirm Password")
			if err != nil {
				return err
			}
			if password != confirmPassword {
				return errors.New("passwords didn't match")
			}
			if cfs.DoesContainMachineID(password) {
				fmt.Println("Note: Password contains machine ID")
			}
			err = cfs.EncryptFile(password)
			if err != nil {
				return err
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
			isEncrypted, err := cfs.IsFileEncrypted()
			if err != nil {
				return err
			}
			if !isEncrypted {
				return errors.New("config file is not encrypted")
			}
			password, err := cli.readPassword("Password")
			if err != nil {
				return err
			}
			err = cfs.DecryptFile(password)
			if err != nil {
				return err
			}
			fmt.Println("Config file decrypted successfully")
			return nil
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yml", "config file path")
	rootCmd.AddCommand(runCmd, configCmd)
	configCmd.AddCommand(configEncryptCmd, configDecryptCmd)
	return rootCmd.Execute()
}

func (cli *cliController) readPassword(prompt string) (string, error) {
	fmt.Print(prompt + ": ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if len(passwordBytes) == 0 {
		err = errors.New("password is empty")
	}
	fmt.Println()
	return string(passwordBytes), err
}
