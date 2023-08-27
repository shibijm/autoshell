package utils

import (
	"fmt"
	"os"
)

func ExitWithError(err error) {
	fmt.Printf("Error: %s\n", err)
	os.Exit(1)
}

func ExitWithWrappedError(err error, message string) {
	ExitWithError(WrapError(err, message))
}

func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}
