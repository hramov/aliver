package executor

import (
	"context"
	"log"
	"os"
	"os/exec"
)

// Execute runs a script command in a separate process.
// It takes a context and the script command as input.
// It returns an error if there was a problem executing the script.
func Execute(ctx context.Context, script string) error {
	done := false
	// Create a new exec.Cmd instance with the script command.
	cmd := &exec.Cmd{
		Path:   script,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	// Log the command being executed.
	log.Println("executing command ", cmd)
	// Start the command in a separate process.
	err := cmd.Start()
	if err != nil {
		return err
	}
	// Goroutine to handle cancellation of the command.
	go func() {
		<-ctx.Done()
		if !done {
			err = cmd.Cancel()
			if err != nil {
				log.Printf("cannot cancel process: %v\n", err)
			}
		}
	}()
	// Wait for the command to finish executing.
	err = cmd.Wait()
	if err != nil {
		return err
	}
	done = true
	return nil
}
