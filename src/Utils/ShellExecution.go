package Utils

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type CommandResult struct {
	ExitCode int
	Output   string
}

func ExecuteCommand(command string) (CommandResult, error) {
	return ExecuteCommandWithEnv(command, nil)
}

func ExecuteCommandWithEnv(command string, envs map[string]string) (CommandResult, error) {
	trimmed := strings.TrimSpace(command)
	parts := strings.Split(trimmed, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if envs != nil {
		cmd.Env = os.Environ()
		for k, v := range envs {
			cmd.Env = append(cmd.Env, k+"='"+v+"'")
		}
	}

	var result CommandResult
	if err := cmd.Run(); err != nil {
		result.Output = buf.String()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			return result, err
		}
	}
	stdout := buf.String()
	result.Output = stdout
	result.ExitCode = 0

	return result, nil
}
