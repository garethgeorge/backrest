package migrations

import (
	"fmt"
	"os"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

var conflictingEnvVars = []string{
	"RESTIC_PASSWORD",
	"RESTIC_PASSWORD_FILE",
	"RESTIC_PASSWORD_COMMAND",
}

// migration005CheckRepoPasswords detects repos whose restic password may have
// been set incorrectly due to a bug fixed in this release
// (https://github.com/garethgeorge/backrest/issues/1139).
//
// The bug: RESTIC_PASSWORD, RESTIC_PASSWORD_FILE, and RESTIC_PASSWORD_COMMAND
// inherited from the process environment previously took precedence over the
// password configured in Backrest's UI, because WithEnviron() was appended
// after the config password. Repos initialized or used under those conditions
// were encrypted with the env var's password, not the one the user entered.
//
// On first run after upgrading, if any repo has a config password AND one of
// the above env vars is set, this migration returns an error and refuses to
// start, printing instructions for the user.
//
// If ISSUE_1139_FIX_PASSWORDS=1 is set in the environment, the migration
// instead applies the following automatic fixes to each affected repo and
// writes the corrected config to disk before startup continues:
//
//   - RESTIC_PASSWORD: the env var's value is written into repo.Password,
//     replacing the (likely wrong) value the user had entered in the UI.
//
//   - RESTIC_PASSWORD_FILE or RESTIC_PASSWORD_COMMAND: repo.Password is
//     cleared and the env var (name=value) is appended to repo.Env, so
//     restic continues to resolve the password via the same mechanism it
//     was using before. The env var can then be removed from the Backrest
//     process environment, as the password source is now explicit in the
//     repo config.
//
// After this migration runs successfully (whether or not a fix was needed)
// the config version is bumped and it will not run again on future starts.
var migration005CheckRepoPasswords = func(config *v1.Config) error {
	// Find repos that have a password configured in the UI.
	var affectedRepos []*v1.Repo
	for _, repo := range config.Repos {
		if repo.GetPassword() != "" {
			affectedRepos = append(affectedRepos, repo)
		}
	}
	if len(affectedRepos) == 0 {
		return nil
	}

	// Check which conflicting env vars are set.
	var setEnvVars []string
	for _, envVar := range conflictingEnvVars {
		if os.Getenv(envVar) != "" {
			setEnvVars = append(setEnvVars, envVar)
		}
	}
	if len(setEnvVars) == 0 {
		return nil
	}

	// There is a potential conflict. Check if the user has acknowledged it.
	if os.Getenv("ISSUE_1139_FIX_PASSWORDS") != "" {
		for _, repo := range affectedRepos {
			for _, envVar := range setEnvVars {
				val := os.Getenv(envVar)
				switch envVar {
				case "RESTIC_PASSWORD":
					zap.S().Warnf("repo %q: overwriting config password with value of RESTIC_PASSWORD env var (issue 1139 fix)", repo.Id)
					repo.Password = val
				case "RESTIC_PASSWORD_FILE", "RESTIC_PASSWORD_COMMAND":
					zap.S().Warnf("repo %q: clearing config password and adding %s=%s to repo env (issue 1139 fix)", repo.Id, envVar, val)
					repo.Password = ""
					repo.Env = append(repo.Env, envVar+"="+val)
				}
			}
		}
		return nil
	}

	// Block startup with a detailed error message.
	var b strings.Builder
	b.WriteString("IMPORTANT: Backrest detected a potential password conflict affecting your restic repos (see https://github.com/garethgeorge/backrest/issues/1139).\n")
	b.WriteString("\n")
	b.WriteString("What happened:\n")
	b.WriteString("  Backrest was using the wrong password for restic repos. The environment variables listed\n")
	b.WriteString("  below were inherited by Backrest's process and previously took precedence over the password\n")
	b.WriteString("  you configured in the UI. Your repos may have been encrypted with a different password than\n")
	b.WriteString("  what your Backrest config says.\n")
	b.WriteString("\n")
	b.WriteString("Conflicting environment variables found:\n")
	for _, envVar := range setEnvVars {
		fmt.Fprintf(&b, "  - %s\n", envVar)
	}
	b.WriteString("\n")
	b.WriteString("Potentially affected repos:\n")
	for _, repo := range affectedRepos {
		fmt.Fprintf(&b, "  - %s\n", repo.Id)
	}
	b.WriteString("\n")
	b.WriteString("How to fix:\n")
	b.WriteString("  Rerun Backrest with ISSUE_1139_FIX_PASSWORDS=1 set in the environment.\n")
	b.WriteString("  Backrest will automatically update each affected repo's config as follows,\n")
	b.WriteString("  then write the corrected config to disk and start normally:\n")
	b.WriteString("\n")
	b.WriteString("    RESTIC_PASSWORD:         the env var's value is written into the repo's\n")
	b.WriteString("                             password field, replacing the value entered in the UI.\n")
	b.WriteString("\n")
	b.WriteString("    RESTIC_PASSWORD_FILE /   the repo's password field is cleared and the env var\n")
	b.WriteString("    RESTIC_PASSWORD_COMMAND: (name=value) is appended to the repo's env list, so\n")
	b.WriteString("                             restic continues to resolve the password the same way.\n")
	b.WriteString("\n")
	b.WriteString("  Once the fix has been applied you can remove the conflicting environment variable\n")
	b.WriteString("  from your Backrest process environment — the password source will be stored\n")
	b.WriteString("  explicitly in the repo config from that point on.\n")

	return fmt.Errorf("%s", b.String())
}
