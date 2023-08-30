package services

import (
	"autoshell/core/entities"
	"autoshell/core/ports"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type variable struct {
	name  string
	value string
}

type runner struct {
	config               *entities.Config
	variables            *[]*variable
	failedCommands       []string
	ignoredErrorCodes    []int
	logFilePath          string
	logFileContentBuffer string
	reporters            []map[string]string
	logReplacements      map[*regexp.Regexp]string
}

func NewRunner(config *entities.Config) ports.Runner {
	return &runner{config, &[]*variable{}, []string{}, []int{}, "", "", []map[string]string{}, map[*regexp.Regexp]string{regexp.MustCompile(`((?:rclone:)?:(?:storj|sftp),).*?(:\S)`): "$1***$2"}}
}

func (r *runner) Run(workflow string, args []string) error {
	r.logSeparator()
	start := time.Now()
	r.logln("Started at %s", start.Format(time.RFC3339Nano))
	err := r.runInstruction("runWorkflow "+workflow, parseArgVars(args))
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
	r.logSeparator()
	return err
}

func (r *runner) runInstruction(instruction string, locals *[]*variable) error {
	if len(instruction) == 0 || instruction[0] == '#' {
		return nil
	}
	tokens := r.tokenise(instruction, locals)
	action := tokens[0]
	args := tokens[1:]
	var err error
	switch action {
	case "runWorkflow":
		if err = checkArgsMin(args, 1); err != nil {
			break
		}
		workflow := args[0]
		instructions, ok := r.config.Workflows[workflow]
		if !ok {
			err = fmt.Errorf("workflow '%s' not found", workflow)
			break
		}
		newLocals := &[]*variable{}
		for _, variables := range []*[]*variable{locals, parseArgVars(args[1:])} {
			for _, v := range *variables {
				setVariable(newLocals, v.name, v.value)
			}
		}
		for _, instruction := range strings.Split(instructions, "\n") {
			err = r.runInstruction(instruction, newLocals)
			if err != nil {
				return err
			}
		}
	case "setEnvVar":
		if err = checkArgsExact(args, 2); err != nil {
			break
		}
		err = os.Setenv(args[0], args[1])
	case "setGlobalVar":
		if err = checkArgsExact(args, 2); err != nil {
			break
		}
		setVariable(r.variables, args[0], args[1])
	case "setLocalVar":
		if err = checkArgsExact(args, 2); err != nil {
			break
		}
		setVariable(locals, args[0], args[1])
	case "runCommand":
		if err = checkArgsMin(args, 2); err != nil {
			break
		}
		commandID := args[0]
		silent2 := commandID == "--"
		silent1 := silent2 || commandID == "-"
		cmd := exec.Command(args[1], args[2:]...)
		if !silent1 {
			r.logSeparator()
			r.logln("Command ID: %s", commandID)
		}
		if !silent2 {
			r.logSeparator()
		}
		var err error
		if r.logFilePath != "" {
			var output []byte
			output, err = cmd.CombinedOutput()
			if !silent2 {
				r.log(string(output))
			}
		} else {
			if !silent2 {
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
			}
			err = cmd.Run()
		}
		if err != nil && !silent1 {
			if exitError, ok := err.(*exec.ExitError); ok {
				var ignoreError bool
				for _, ignoredErrorCode := range r.ignoredErrorCodes {
					if exitError.ProcessState.ExitCode() == ignoredErrorCode {
						ignoreError = true
						break
					}
				}
				if ignoreError {
					break
				}
			}
			r.logln("%s failed: %s", action, err)
			r.failedCommands = append(r.failedCommands, commandID)
		}
	case "setLogFile":
		if err = checkArgsExact(args, 1); err != nil {
			break
		}
		err = appendToFile(args[0], r.logFileContentBuffer)
		if err == nil {
			r.logFilePath = args[0]
			r.logFileContentBuffer = ""
		}
	case "addReporter":
		if err = checkArgsMin(args, 1); err != nil {
			break
		}
		reporterType := args[0]
		switch reporterType {
		case "uptimeKuma":
			if err = checkArgsExact(args, 2); err != nil {
				break
			}
			r.reporters = append(r.reporters, map[string]string{"type": reporterType, "endpoint": args[1]})
		default:
			err = fmt.Errorf("reporter type '%s' is not supported", reporterType)
		}
	case "setIgnoredErrorCodes":
		if err = checkArgsExact(args, 1); err != nil {
			break
		}
		err = json.Unmarshal([]byte(args[0]), &r.ignoredErrorCodes)
	case "print":
		r.logln(strings.Join(args, " "))
	case "shiftArgVars":
		if err = checkArgsExact(args, 0); err != nil {
			break
		}
		v := getVariable(locals, "@")
		if v != nil {
			tokens := r.tokenise(v.value, &[]*variable{})
			if len(tokens) <= 1 {
				setVariable(locals, "@", "")
				setVariable(locals, "1", "")
			} else {
				argVars := parseArgVars(tokens[1:])
				for _, v := range *argVars {
					setVariable(locals, v.name, v.value)
				}
				setVariable(locals, strconv.Itoa(len(tokens)), "")
			}
		}
	default:
		err = errors.New("unrecognised action")
	}
	if err != nil {
		err = fmt.Errorf("action '%s' failed: %w", action, err)
	}
	return err
}

func (r *runner) tokenise(input string, variables *[]*variable) []string {
	allVariables := append(append([]*variable{}, *variables...), *r.variables...)
	sort.SliceStable(allVariables, func(i, j int) bool {
		return len(allVariables[i].name) > len(allVariables[j].name)
	})
	for _, v := range allVariables {
		input = strings.ReplaceAll(input, "$"+v.name, v.value)
	}
	input = os.ExpandEnv(input)
	var tokens []string
	var currentToken string
	var withinQuotes rune
	var lastChar rune
	for _, char := range input {
		if (char == '"' || char == '\'') && (lastChar == 0 || lastChar == ' ' || withinQuotes != 0) {
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
		lastChar = char
	}
	if currentToken != "" {
		tokens = append(tokens, currentToken)
	}
	return tokens
}

func (r *runner) report(elapsedSeconds int, errSummary string, errDetail string) {
	for _, reporter := range r.reporters {
		var err error
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
			_, err = http.Get(fmt.Sprintf("%s?status=%s&msg=%s&ping=%d", endpoint, status, url.QueryEscape(msg), elapsedSeconds))
		default:
			err = errors.New("unsupported reporter type")
		}
		if err != nil {
			r.logln("Reporter with type '%s' failed: %s", reporter["type"], err)
		}
	}
}

func (r *runner) log(format string, a ...any) {
	text := fmt.Sprintf(format, a...)
	for re, replacement := range r.logReplacements {
		text = re.ReplaceAllString(text, replacement)
	}
	fmt.Print(text)
	if r.logFilePath != "" {
		err := appendToFile(r.logFilePath, text)
		if err != nil {
			fmt.Printf("Failed to write to log file: %s\n", err)
		}
	} else {
		r.logFileContentBuffer += text
	}
}

func (r *runner) logln(format string, a ...any) {
	r.log(format+"\n", a...)
}

func (r *runner) logSeparator() {
	r.logln(strings.Repeat("-", 80))
}

func parseArgVars(args []string) *[]*variable {
	variables := &[]*variable{}
	for i, arg := range args {
		setVariable(variables, strconv.Itoa(i+1), arg)
	}
	if len(args) > 0 {
		setVariable(variables, "@", strings.Join(args, " "))
	}
	return variables
}

func getVariable(variables *[]*variable, name string) *variable {
	for _, v := range *variables {
		if v.name == name {
			return v
		}
	}
	return nil
}

func setVariable(variables *[]*variable, name string, value string) {
	v := getVariable(variables, name)
	if v != nil {
		v.value = value
	} else {
		*variables = append(*variables, &variable{name, value})
	}
}

func checkArgs(args []string, expected int, compare func(argsLength int, expected int) (bool, string)) error {
	argsLength := len(args)
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

func appendToFile(filePath string, text string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(text)
	return err
}
