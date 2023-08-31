package utils

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/term"
)

func ReadInputHidden(prompt string) (string, error) {
	fmt.Print(prompt + ": ")
	inputBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err == nil && len(inputBytes) == 0 {
		err = errors.New("input is empty")
	}
	if err != nil {
		return "", err
	}
	input := string(inputBytes)
	return input, nil
}
