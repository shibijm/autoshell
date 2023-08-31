package utils

import "fmt"

func checkArgs(args []string, expected int, compare func(argsLength int, expected int) (bool, string)) error {
	argsLength := len(args)
	failed, expectedText := compare(argsLength, expected)
	if failed {
		return fmt.Errorf("invalid number of args, %s %d, received %d", expectedText, expected, argsLength)
	}
	return nil
}

func CheckArgsExact(args []string, expected int) error {
	return checkArgs(args, expected, func(argsLength int, expected int) (bool, string) {
		return argsLength != expected, "expected"
	})
}

func CheckArgsMin(args []string, expected int) error {
	return checkArgs(args, expected, func(argsLength int, expected int) (bool, string) {
		return argsLength < expected, "expected at least"
	})
}
