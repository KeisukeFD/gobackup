package Utils

import (
	"bufio"
	"fmt"
	"io"
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

	if envs != nil {
		cmd.Env = os.Environ()
		for k, v := range envs {
			cmd.Env = append(cmd.Env, k+"="+v+"")
		}
	}
	var result CommandResult
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))

	err := cmd.Start()
	if err != nil {
		result.Output = scanner.Text()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			return result, err
		}
	}

	for scanner.Scan() {
		m := scanner.Text()
		result.Output += m
		fmt.Println(m)
	}
	cmd.Wait()

	result.ExitCode = 0

	return result, nil
}
