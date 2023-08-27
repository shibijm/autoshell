package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"autoshell/utils"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type runner struct {
	config         *entities.Config
	variables      map[string]string
	failedCommands []string
}

func NewRunner(config *entities.Config) ports.Runner {
	return &runner{config, map[string]string{}, []string{}}
}

func (r *runner) Run(workflowName string, args []string) error {
	r.variables = map[string]string{}
	r.failedCommands = []string{}
	for i, arg := range args {
		r.variables[strconv.Itoa(i+1)] = arg
	}
	r.variables["@"] = strings.Join(args, " ")
	r.logSeparator()
	start := time.Now()
	r.logln("Started at %s", start.Format(time.RFC3339Nano))
	err := r.runInstruction("runWorkflow " + workflowName)
	end := time.Now()
	elapsedSeconds := int(math.Round(end.Sub(start).Seconds()))
	r.logSeparator()
	r.logln("Ended at %s (took %d seconds)", end.Format(time.RFC3339Nano), elapsedSeconds)
	errCausedAbort := err != nil
	if len(r.failedCommands) > 0 {
		failedCommandsErr := fmt.Errorf("failed commands: %s", strings.Join(r.failedCommands, ", "))
		if err != nil {
			err = fmt.Errorf("%w; %w", err, failedCommandsErr)
		} else {
			err = failedCommandsErr
		}
	}
	var errSummary string
	var errDetail string
	if err != nil {
		if errCausedAbort {
			errSummary = "Aborted with errors"
		} else {
			errSummary = "Finished with errors"
		}
		runes := []rune(err.Error())
		runes[0] = unicode.ToUpper(runes[0])
		errDetail = string(runes)
		r.logSeparator()
		r.logln("%s", errDetail)
		err = fmt.Errorf("runner %s", strings.ToLower(errSummary))
	}
	r.report(elapsedSeconds, errSummary, errDetail)
	return err
}

func (r *runner) runInstruction(instruction string) error {
	if len(instruction) == 0 || instruction[0] == '#' {
		return nil
	}
	tokens := r.tokenise(instruction)
	action := tokens[0]
	var err error
	switch action {
	case "runWorkflow":
		if err = checkArgsExact(tokens, 1); err != nil {
			break
		}
		workflowName := tokens[1]
		instructions, ok := r.config.Workflows[workflowName]
		if !ok {
			err = fmt.Errorf("workflow '%s' not found", workflowName)
			break
		}
		for _, instruction := range strings.Split(instructions, "\n") {
			err = r.runInstruction(instruction)
			if err != nil {
				return err
			}
		}
	case "setVar":
		if err = checkArgsExact(tokens, 2); err != nil {
			break
		}
		r.variables[tokens[1]] = tokens[2]
	case "setEnvVar":
		if err = checkArgsExact(tokens, 2); err != nil {
			break
		}
		err = os.Setenv(tokens[1], tokens[2])
	case "runCommand":
		if err = checkArgsMin(tokens, 2); err != nil {
			break
		}
		commandID := tokens[1]
		silent := commandID == "-"
		cmd := exec.Command(tokens[2], tokens[3:]...)
		if !silent {
			r.logSeparator()
			r.logln("Command: %s", commandID)
			r.logSeparator()
		}
		var err error
		if r.config.LogFilePath != "" {
			var output []byte
			output, err = cmd.CombinedOutput()
			if !silent {
				r.log(string(output))
			}
		} else {
			if !silent {
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
			}
			err = cmd.Run()
		}
		if err != nil && !silent {
			r.logln("%s failed: %s", action, err)
			r.failedCommands = append(r.failedCommands, commandID)
		}
	default:
		err = errors.New("unrecognised action")
	}
	if err != nil {
		err = utils.WrapError(err, fmt.Sprintf("action '%s' failed", action))
	}
	return err
}

func (r *runner) tokenise(input string) []string {
	for k, v := range r.variables {
		input = strings.ReplaceAll(input, "$"+k, v)
	}
	input = os.ExpandEnv(input)
	var tokens []string
	var currentToken string
	var withinQuotes rune
	for _, char := range input {
		if char == '"' || char == '\'' {
			if withinQuotes == 0 {
				withinQuotes = char
			} else if withinQuotes == char {
				withinQuotes = 0
			} else {
				currentToken += string(char)
			}
		} else if char == ' ' && withinQuotes == 0 {
			if currentToken != "" {
				tokens = append(tokens, currentToken)
				currentToken = ""
			}
		} else {
			currentToken += string(char)
		}
	}
	if currentToken != "" {
		tokens = append(tokens, currentToken)
	}
	return tokens
}

func checkArgs(args []string, expected int, compare func(argsLength int, expected int) (bool, string)) error {
	argsLength := len(args) - 1
	failed, expectedText := compare(argsLength, expected)
	if failed {
		return fmt.Errorf("invalid number of args, %s %d, received %d", expectedText, expected, argsLength)
	}
	return nil
}

func checkArgsExact(args []string, expected int) error {
	return checkArgs(args, expected, func(argsLength int, expected int) (bool, string) {
		return argsLength != expected, "expected"
	})
}

func checkArgsMin(args []string, expected int) error {
	return checkArgs(args, expected, func(argsLength int, expected int) (bool, string) {
		return argsLength < expected, "expected at least"
	})
}

func (r *runner) report(elapsedSeconds int, errSummary string, errDetail string) {
	for _, reporter := range r.config.Reporters {
		var reporterErr error
		switch reporter["type"] {
		case "uptimeKuma":
			endpoint := reporter["endpoint"]
			var msg string
			if errSummary != "" {
				msg = errSummary
			} else {
				msg = "Finished successfully"
			}
			http.Get(fmt.Sprintf("%s?status=up&msg=%s&ping=%d", endpoint, url.QueryEscape(msg), elapsedSeconds))
			var status string
			if errDetail != "" {
				status = "down"
				msg = errDetail
			} else {
				status = "up"
			}
			_, reporterErr = http.Get(fmt.Sprintf("%s?status=%s&msg=%s&ping=%d", endpoint, status, url.QueryEscape(msg), elapsedSeconds))
		default:
			reporterErr = errors.New("unsupported reporter")
		}
		if reporterErr != nil {
			r.logln("Failed to report with reporter '%s' - %s", reporter["type"], reporterErr)
		}
	}
}

func (r *runner) log(format string, a ...any) {
	text := fmt.Sprintf(format, a...)
	fmt.Print(text)
	if r.config.LogFilePath != "" {
		file, err := os.OpenFile(r.config.LogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		_, err = file.WriteString(text)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (r *runner) logln(format string, a ...any) {
	r.log(format+"\n", a...)
}

func (r *runner) logSeparator() {
	r.logln(strings.Repeat("-", 80))
}
