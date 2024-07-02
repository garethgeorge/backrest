package hook

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestHookCommandInDefaultShell(t *testing.T) {
	hook := Hook(v1.Hook{
		Conditions: []v1.Hook_Condition{v1.Hook_CONDITION_SNAPSHOT_START},
		Action: &v1.Hook_ActionCommand{
			ActionCommand: &v1.Hook_Command{
				Command: "exit 2",
			},
		},
	})

	err := hook.Do(v1.Hook_CONDITION_SNAPSHOT_START, struct{}{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %v", err.(*exec.ExitError).ExitCode())
	}
}

func TestHookCommandInBashShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	hook := Hook(v1.Hook{
		Conditions: []v1.Hook_Condition{v1.Hook_CONDITION_SNAPSHOT_START},
		Action: &v1.Hook_ActionCommand{
			ActionCommand: &v1.Hook_Command{
				Command: `#!/bin/bash
counter=0
# Start a while loop that will run until the counter is equal to 10
while [ $counter -lt 10 ]; do
  ((counter++))
done
exit $counter`,
			},
		},
	})

	err := hook.Do(v1.Hook_CONDITION_SNAPSHOT_START, struct{}{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.(*exec.ExitError).ExitCode() != 10 {
		t.Fatalf("expected exit code 3, got %v", err.(*exec.ExitError).ExitCode())
	}
}

func TestCommandHookErrorHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on windows")
	}

	hook := Hook(v1.Hook{
		Conditions: []v1.Hook_Condition{
			v1.Hook_CONDITION_SNAPSHOT_START,
		},
		Action: &v1.Hook_ActionCommand{
			ActionCommand: &v1.Hook_Command{
				Command: "exit 1",
			},
		},
		OnError: v1.Hook_ON_ERROR_CANCEL,
	})

	err := applyHookErrorPolicy(hook.OnError, hook.Do(v1.Hook_CONDITION_SNAPSHOT_START, struct{}{}, &bytes.Buffer{}))
	if err == nil {
		t.Fatal("expected error")
	}
	var cancelErr *HookErrorRequestCancel
	if !errors.As(err, &cancelErr) {
		t.Fatalf("expected HookErrorRequestCancel, got %v", err)
	}
}
