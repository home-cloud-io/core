package execute

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
)

// ExecuteCommand executes a command and and prints the output to stdout.
// It will not return until the command has completed or the context is cancelled.
//
// NOTE: If you want the command to continue even if the parent is killed (e.g. through a signal like SIGINT) you will
// want to launch the command in a new process group so that signals are not sent to the child process.
//
// 	cmd.SysProcAttr = &syscall.SysProcAttr{
//		Setpgid: true,
//	}
func ExecuteCommand(ctx context.Context, cmd *exec.Cmd) error {
	// create a pipe for the output
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	// create a pipe for the error output
	cmdErrReader, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// scanner for output
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	// scanner for error output
	scannerErr := bufio.NewScanner(cmdErrReader)
	go func() {
		for scannerErr.Scan() {
			fmt.Println(scannerErr.Text())
		}
	}()

	// watch for done signal and kill process if received
	go func() {
		<-ctx.Done()
		var err error
		// if the command was launched as a new process group so that signals (e.g. SIGINT) are not sent to the child process
		// then we should release the process instead of killing it
		if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
			err = cmd.Process.Release()
		} else {
			err = cmd.Process.Kill()
		}
		if err != nil {
			// only error if not closed by user
			if err.Error() != "signal: killed" && err.Error() != "os: process already finished" {
				fmt.Println(err.Error())
			}
		}
	}()

	// start the command
	err = cmd.Start()
	if err != nil {
		return err
	}

	// wait for completion
	err = cmd.Wait()
	if err != nil {
		// only error if not closed by user
		if err.Error() != "signal: killed" && err.Error() != "os: process already finished" {
			return err
		}
	}

	return nil
}

// ExecuteCommandAndRelease will execute the given command and release all associated resources so that it will
// continue to run even if the caller is terminated.
//
// NOTE: You probably want to launch the command in a new process group to avoid signals (e.g. SIGINT) being sent to the child process.
//
// 	cmd.SysProcAttr = &syscall.SysProcAttr{
//		Setpgid: true,
//	}
func ExecuteCommandAndRelease(ctx context.Context, cmd *exec.Cmd) error {
	// start the command
	err := cmd.Start()
	if err != nil {
		return err
	}

	// release any associated resources
	err = cmd.Process.Release()
	if err != nil {
		return err
	}

	return nil
}

// ExecuteCommandReturnStdout executes a command and returns the output of stdout as a string.
// It does not print the output to the console. This can be used to get the output of a command.
// It will not return until the command has completed or the context is cancelled.
func ExecuteCommandReturnStdout(ctx context.Context, cmd *exec.Cmd) (output string, err error) {
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
