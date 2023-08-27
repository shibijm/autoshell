package cli

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type CliController struct {
	version       string
	configService ports.ConfigService
	runner        ports.Runner
}

func NewCliController(version string, configService ports.ConfigService, runner ports.Runner) *CliController {
	return &CliController{version, configService, runner}
}

func (cli *CliController) Execute() error {
	var configPath string
	cobra.OnInitialize(func() {
		_, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			fmt.Printf("Error: config file '%s' does not exist\n", configPath)
			os.Exit(1)
		}
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			isEncrypted, err := cli.configService.IsEncryptedConfigFile(configPath)
			if err != nil {
				return err
			}
			var config *entities.Config
			if isEncrypted {
				config, err = cli.configService.ParseConfigFileWithMachineID(configPath)
				if err != nil {
					password, pwdErr := cli.readPassword("Password")
					if pwdErr != nil {
						return pwdErr
					}
					config, err = cli.configService.ParseConfigFileWithPassword(configPath, password)
				}
			} else {
				config, err = cli.configService.ParseConfigFile(configPath)
			}
			if err != nil {
				return err
			}
			return cli.runner.Run(config, args[0])
		},
	}
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Config commands",
	}
	configEncryptCmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt the config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			isEncrypted, err := cli.configService.IsEncryptedConfigFile(configPath)
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
			if cli.configService.ContainsMachineID(password) {
				fmt.Println("Note: Password contains machine ID")
			}
			err = cli.configService.EncryptConfigFile(configPath, password)
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
			isEncrypted, err := cli.configService.IsEncryptedConfigFile(configPath)
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
			err = cli.configService.DecryptConfigFile(configPath, password)
			if err != nil {
				return err
			}
			fmt.Println("Config file decrypted successfully")
			return nil
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path")
	rootCmd.AddCommand(runCmd, configCmd)
	configCmd.AddCommand(configEncryptCmd, configDecryptCmd)
	return rootCmd.Execute()
}

func (cli *CliController) readPassword(prompt string) (string, error) {
	fmt.Print(prompt + ": ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if len(passwordBytes) == 0 {
		err = errors.New("password is empty")
	}
	fmt.Println()
	return string(passwordBytes), err
}
