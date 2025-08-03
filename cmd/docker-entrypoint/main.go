package main

import (
	"fmt"
	"os"
	"os/exec"
)

var dockerEnvVarDefaults = map[string]string{
	"BACKREST_PORT": "0.0.0.0:9898",
}

func main() {
	var defaultedVariables []string
	for key, value := range dockerEnvVarDefaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			defaultedVariables = append(defaultedVariables, fmt.Sprintf("%s=%q", key, value))
		}
	}
	if len(defaultedVariables) > 0 {
		os.Stderr.WriteString("Setting docker defaults for env variables:\n")
		for _, key := range defaultedVariables {
			os.Stderr.WriteString(" - " + key + "\n")
		}
	}
	if len(os.Args) < 2 {
		os.Stderr.WriteString("No command provided to run.\n")
		os.Exit(1)
	}
	subcommand := os.Args[1]
	args := os.Args[2:]
	os.Stderr.WriteString("Running command: " + subcommand + " " + fmt.Sprint(args) + "\n")

	cmd := exec.Command(subcommand, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		panic(err)
	}
}
