package runner

import (
	"autoshell/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Runner struct {
	config           config.Config
	vars             map[string]string
	failedCommands   []string
	ignoredExitCodes []int
	logFilePath      string
	logFileBuffer    strings.Builder
	reporters        []reporter
	httpClient       http.Client
}

type reporter struct {
	kind     string
	endpoint string
}

func New(cfg config.Config) *Runner {
	return &Runner{
		config:     cfg,
		vars:       make(map[string]string),
		httpClient: http.Client{Timeout: 10 * time.Second},
	}
}

func (r *Runner) RunWorkflow(args []string) error {
	start := time.Now()
	r.log("Started at %s", start.Format(time.RFC3339Nano))
	err := r.runAction("runWorkflow", args, map[string]string{}, map[string]string{})
	end := time.Now()
	elapsed := end.Sub(start)
	r.log("Ended at %s after %dms", end.Format(time.RFC3339Nano), elapsed.Milliseconds())
	var errMsgs []string
	if len(r.failedCommands) > 0 {
		errMsgs = append(errMsgs, "Failed commands: "+strings.Join(r.failedCommands, ", "))
	}
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	r.report(elapsed, errMsgs)
	defer r.log("")
	if len(errMsgs) > 0 {
		r.log("%s", strings.Join(errMsgs, "\n"))
		return errors.New("runner failed")
	}
	return nil
}

func (r *Runner) log(format string, args ...any) {
	text := strings.Repeat("-", 80) + "\n"
	if format != "" {
		if line := fmt.Sprintf(format, args...); line != "" {
			text += line + "\n"
		}
	}
	fmt.Print(text)
	if r.logFilePath != "" {
		if err := r.appendToLogFile(text); err != nil {
			fmt.Println("Failed to write to log file: " + err.Error())
		}
	} else {
		r.logFileBuffer.WriteString(text)
	}
}

func (r *Runner) appendToLogFile(text string) error {
	file, err := os.OpenFile(r.logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()
	if _, err = file.WriteString(text); err != nil {
		return fmt.Errorf("write to file: %w", err)
	}
	return nil
}

const maxRespBodyBytes = 8 * 1024 * 1024

func (r *Runner) report(elapsed time.Duration, errMsgs []string) {
	for _, reporter := range r.reporters {
		var err error
		switch reporter.kind {
		case "uptimeKuma":
			var status, msg string
			if len(errMsgs) > 0 {
				status = "down"
				msg = strings.Join(errMsgs, "\n")
			} else {
				status = "up"
				msg = "Finished successfully"
			}
			var resp *http.Response
			resp, err = r.httpClient.Get(fmt.Sprintf("%s?status=%s&msg=%s&ping=%d", reporter.endpoint, status, url.QueryEscape(msg), elapsed.Milliseconds()))
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespBodyBytes))
				err = fmt.Errorf("HTTP %d, %#v, %s", resp.StatusCode, resp.Header, string(bodyBytes))
			}
		default:
			err = errors.New("invalid kind")
		}
		if err != nil {
			r.log("Reporter %q failed: %s", reporter.kind, err)
		}
	}
}

func (r *Runner) runAction(action string, args []string, vars map[string]string, modifiers map[string]string) error {
	if action == "" || action[0] == '#' {
		return nil
	}
	var err error
	switch action {
	case "runWorkflow":
		if err = checkArgsMin(args, 1); err != nil {
			break
		}
		workflow := args[0]
		instructions, ok := r.config.Workflows[workflow]
		if !ok {
			err = fmt.Errorf("workflow %q not found", workflow)
			break
		}
		args := args[1:]
		vars := maps.Clone(vars)
	instLoop:
		for instruction := range strings.SplitSeq(instructions, "\n") {
			tokens := r.tokenise(instruction, args, vars)
			if len(tokens) == 0 {
				continue
			}
			action := tokens[0]
			modifiers := maps.Clone(modifiers)
			actionSplit := strings.Split(action, "!")
			if len(actionSplit) == 2 {
				action = actionSplit[0]
				for modifier := range strings.SplitSeq(actionSplit[1], ",") {
					switch modifier {
					case "W":
						if runtime.GOOS != "windows" {
							continue instLoop
						}
					case "L":
						if runtime.GOOS != "linux" {
							continue instLoop
						}
					}
					k, v, found := strings.Cut(modifier, "=")
					if !found {
						v = "true"
					}
					modifiers[k] = v
				}
			}
			if action == "shiftArgs" {
				argsLen := len(args)
				if argsLen > 0 {
					for i := range argsLen - 1 {
						args[i] = args[i+1]
					}
					args = args[:argsLen-1]
				}
				continue
			}
			if err = r.runAction(action, tokens[1:], vars, modifiers); err != nil {
				break
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
		r.vars[args[0]] = args[1]
	case "setLocalVar":
		if err = checkArgsExact(args, 2); err != nil {
			break
		}
		vars[args[0]] = args[1]
	case "runCommand":
		if err = checkArgsMin(args, 2); err != nil {
			break
		}
		commandId := args[0]
		retries := 0
		if retriesStr, ok := modifiers["retries"]; ok {
			retries, _ = strconv.Atoi(retriesStr)
		}
		var cmdErr error
		for i := 0; i <= retries; i++ {
			if i > 0 {
				r.log("Retrying (%d/%d)", i, retries)
			} else if modifiers["hideCommandId"] != "true" {
				r.log("Command ID: %s", commandId)
			}
			cmd := exec.Command(args[1], args[2:]...) //nolint:gosec
			if r.logFilePath != "" {
				var output []byte
				output, cmdErr = cmd.CombinedOutput()
				r.log("%s", strings.TrimSuffix(string(output), "\n"))
			} else {
				r.log("")
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
				cmdErr = cmd.Run()
			}
			if cmdErr == nil {
				break
			}
			if exitError, ok := errors.AsType[*exec.ExitError](cmdErr); ok {
				if slices.Contains(r.ignoredExitCodes, exitError.ExitCode()) {
					break
				}
			}
			r.log("%s failed: %s", action, cmdErr)
			if i < retries {
				continue
			}
			if modifiers["ignoreFailures"] != "true" {
				r.failedCommands = append(r.failedCommands, commandId)
			}
		}
	case "setLogFile":
		if err = checkArgsExact(args, 1); err != nil {
			break
		}
		r.logFilePath = args[0]
		err = r.appendToLogFile(r.logFileBuffer.String())
		if err != nil {
			r.logFilePath = ""
		} else {
			r.logFileBuffer.Reset()
		}
	case "addReporter":
		if err = checkArgsExact(args, 2); err != nil {
			break
		}
		r.reporters = append(r.reporters, reporter{kind: args[0], endpoint: args[1]})
	case "setIgnoredExitCodes":
		if err = checkArgsExact(args, 1); err != nil {
			break
		}
		err = json.Unmarshal([]byte(args[0]), &r.ignoredExitCodes)
	case "print":
		r.log("%s", strings.Join(args, " "))
	default:
		err = errors.New("invalid action")
	}
	if err != nil {
		err = fmt.Errorf("%s: %w", action, err)
	}
	return err
}

func checkArgsExact(args []string, expected int) error {
	if len(args) != expected {
		return fmt.Errorf("invalid number of args, expected %d, received %d", expected, len(args))
	}
	return nil
}

func checkArgsMin(args []string, expected int) error {
	if len(args) < expected {
		return fmt.Errorf("invalid number of args, expected at least %d, received %d", expected, len(args))
	}
	return nil
}

const varPrefix = "$"

func (r *Runner) tokenise(instruction string, args []string, vars map[string]string) []string {
	argVars := make(map[string]string)
	argsQuoted := make([]string, len(args))
	for i, arg := range args {
		argVars[strconv.Itoa(i+1)] = arg
		argsQuoted[i] = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
	}
	argVars["@"] = strings.Join(argsQuoted, " ")
	instruction = os.Expand(instruction, func(k string) string {
		if k == varPrefix {
			return varPrefix
		}
		for _, varMap := range []map[string]string{argVars, vars, r.vars} {
			if v, ok := varMap[k]; ok {
				return v
			}
		}
		return os.Getenv(k)
	})
	var tokens []string
	var currentToken strings.Builder
	var forceAppend bool
	appendCurrentToken := func() {
		if currentToken.Len() > 0 || forceAppend {
			tokens = append(tokens, currentToken.String())
			currentToken.Reset()
			forceAppend = false
		}
	}
	var escaped, inSingleQuote, inDoubleQuote bool
	for _, char := range instruction {
		if escaped {
			escaped = false
			if char == '\\' || char == '\'' || char == '"' {
				currentToken.WriteRune(char)
				continue
			}
			currentToken.WriteRune('\\')
		}
		switch char {
		case '\\':
			escaped = true
		case '\'':
			if inDoubleQuote {
				currentToken.WriteRune(char)
			} else {
				inSingleQuote = !inSingleQuote
			}
			forceAppend = true
		case '"':
			if inSingleQuote {
				currentToken.WriteRune(char)
			} else {
				inDoubleQuote = !inDoubleQuote
			}
			forceAppend = true
		case ' ':
			if inSingleQuote || inDoubleQuote {
				currentToken.WriteRune(char)
			} else {
				appendCurrentToken()
			}
		default:
			currentToken.WriteRune(char)
		}
	}
	appendCurrentToken()
	return tokens
}
