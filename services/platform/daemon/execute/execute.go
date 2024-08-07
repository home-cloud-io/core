package execute

import (
	"bufio"
	"context"
	"os/exec"
)

// Execute runs a command and returns the output of stdout as a string.
// It will not return until the command has completed or the context is cancelled.
func Execute(ctx context.Context, cmd *exec.Cmd) (output string, err error) {
	// create a pipe for the output
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	// scanner for output
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			if output != "" {
				output += "\n"
			}
			output += scanner.Text()
		}
	}()

	// start the command
	err = cmd.Start()
	if err != nil {
		return "", err
	}

	// watch for done signal and kill process if received
	go func() {
		<-ctx.Done()
		_ = cmd.Process.Kill()
	}()

	// wait for completion
	err = cmd.Wait()
	if err != nil {
		// only error if not closed by user
		if err.Error() != "signal: killed" && err.Error() != "os: process already finished" {
			return output, err
		}
	}

	return output, nil
}
