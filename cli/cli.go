package cli

import (
	"autoshell/config"
	"autoshell/runner"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "0.0.0-dev"

func Run() error {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yml", "config file path")
	rootCmd.AddCommand(runCmd, encryptCmd, decryptCmd)
	return rootCmd.Execute()
}

var configPath string

var rootCmd = &cobra.Command{
	Use:           "autoshell",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version: version,
}

var runCmd = &cobra.Command{
	Use:   "run <workflow> [args...]",
	Short: "Run a workflow",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Get(configPath, readPasswordOnce)
		if err != nil {
			return err
		}
		return runner.New(cfg).RunWorkflow(args)
	},
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt the config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		message, err := config.Encrypt(configPath, readPasswordTwice)
		if err != nil {
			return err
		}
		if message != "" {
			fmt.Println(message)
		}
		fmt.Println("Config file encrypted successfully")
		return nil
	},
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt the config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Decrypt(configPath, readPasswordOnce); err != nil {
			return err
		}
		fmt.Println("Config file decrypted successfully")
		return nil
	},
}

func readPasswordOnce() (string, error) {
	return readPassword(false)
}

func readPasswordTwice() (string, error) {
	return readPassword(true)
}

func readPassword(requiresConfirmation bool) (string, error) {
	if password := os.Getenv("AUTOSHELL_PASSWORD"); password != "" {
		return password, nil
	}
	password, err := readHiddenInput("Password")
	if err != nil {
		return "", err
	}
	if !requiresConfirmation {
		return password, nil
	}
	confirmationPassword, err := readHiddenInput("Confirm Password")
	if err != nil {
		return "", err
	}
	if confirmationPassword != password {
		return "", errors.New("the two passwords don't match")
	}
	return password, nil
}

func readHiddenInput(prompt string) (string, error) {
	fmt.Print(prompt + ": ")
	input, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec
	fmt.Println()
	if err != nil {
		return "", err
	}
	if len(input) == 0 {
		return "", errors.New("input is empty")
	}
	return string(input), nil
}
