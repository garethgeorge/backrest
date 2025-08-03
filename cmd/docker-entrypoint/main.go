//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

var dockerEnvVarDefaults = map[string]string{
	"BACKREST_PORT": "0.0.0.0:9898",
	"PUID":          "1000",
	"PGID":          "1000",
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

	// Setup user and group
	uid, gid, err := setupUserAndGroup()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to setup user/group: %v\n", err))
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		os.Stderr.WriteString("No command provided to run.\n")
		os.Exit(1)
	}

	if uid != 0 {
		os.Stderr.WriteString(fmt.Sprintf("Switching to user with UID=%d, GID=%d\n", uid, gid))
	}
	os.Stderr.WriteString("Running command: " + os.Args[1] + " " + fmt.Sprint(os.Args[2:]) + "\n")

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set the user and group for the command if not root
	if uid != 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		panic(err)
	}
}

// setupUserAndGroup handles PUID/PGID setup
func setupUserAndGroup() (int, int, error) {
	puidStr := os.Getenv("PUID")
	pgidStr := os.Getenv("PGID")

	// Default to running as root if PUID is empty or 0
	if puidStr == "" || puidStr == "0" {
		return 0, 0, nil
	}

	puid, err := strconv.Atoi(puidStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid PUID: %v", err)
	}

	pgid, err := strconv.Atoi(pgidStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid PGID: %v", err)
	}

	username := "abc"
	groupname := "abc"

	// Check if user already exists
	if !userExists(username) {
		// Create group if it doesn't exist
		if !groupExists(groupname) {
			if err := createGroup(groupname, pgid); err != nil {
				return 0, 0, fmt.Errorf("failed to create group %s: %v", groupname, err)
			}
			os.Stderr.WriteString(fmt.Sprintf("Created group %s with GID %d\n", groupname, pgid))
		}

		// Create user
		if err := createUser(username, groupname, puid); err != nil {
			return 0, 0, fmt.Errorf("failed to create user %s: %v", username, err)
		}
		os.Stderr.WriteString(fmt.Sprintf("Created user %s with UID %d\n", username, puid))
	}

	return puid, pgid, nil
}

// userExists checks if a user exists by name
func userExists(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

// groupExists checks if a group exists by running getent group
func groupExists(groupname string) bool {
	cmd := exec.Command("getent", "group", groupname)
	return cmd.Run() == nil
}

// createGroup creates a group with the specified GID
func createGroup(groupname string, gid int) error {
	cmd := exec.Command("groupadd", "-g", strconv.Itoa(gid), groupname)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createUser creates a user with the specified UID and group
func createUser(username, groupname string, uid int) error {
	cmd := exec.Command("useradd", "-m", "-u", strconv.Itoa(uid), "-g", groupname, username)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
