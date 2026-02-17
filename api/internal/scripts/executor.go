package scripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// Executor runs Python scripts that wrap samba-tool and other CLI utilities.
// Scripts output JSON to stdout and errors to stderr.
type Executor struct {
	scriptsPath string
	timeout     time.Duration
}

func NewExecutor(scriptsPath string) *Executor {
	return &Executor{
		scriptsPath: scriptsPath,
		timeout:     30 * time.Second,
	}
}

// Run executes a Python script with the given action and arguments.
// Returns the parsed JSON output.
func (e *Executor) Run(ctx context.Context, script, action string, args ...string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmdArgs := append([]string{e.scriptPath(script), action}, args...)
	cmd := exec.CommandContext(ctx, "python3", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("executing script",
		"script", script,
		"action", action,
		"args", args,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("script %s %s: %w: %s", script, action, err, stderr.String())
	}

	return json.RawMessage(stdout.Bytes()), nil
}

// RunWithInput executes a Python script with JSON input piped to stdin.
func (e *Executor) RunWithInput(ctx context.Context, script, action string, input any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	inputData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", e.scriptPath(script), action)
	cmd.Stdin = bytes.NewReader(inputData)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("executing script with input",
		"script", script,
		"action", action,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("script %s %s: %w: %s", script, action, err, stderr.String())
	}

	return json.RawMessage(stdout.Bytes()), nil
}

func (e *Executor) scriptPath(name string) string {
	return fmt.Sprintf("%s/%s", e.scriptsPath, name)
}
