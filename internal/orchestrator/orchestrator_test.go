package orchestrator

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
)

func TestAutoInitializeRepos(t *testing.T) {
	t.Parallel()

	configMgr := &config.ConfigManager{
		Store: &config.MemoryStore{
			Config: &v1.Config{
				Version:  4,
				Instance: "test-instance",
				Repos: []*v1.Repo{
					{
						Id:  "test",
						Uri: t.TempDir(),
						Flags: []string{
							"--no-cache",
							"--insecure-no-password",
						},
						AutoInitialize: true,
					},
				},
			},
		},
	}

	resticBin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("failed to find or install restic binary: %v", err)
	}

	_, err = NewOrchestrator(resticBin, configMgr, nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	if err != nil {
		t.Fatalf("failed to construct orchestrator: %v", err)
	}

	newConfig, _ := configMgr.Get()

	if newConfig.Repos[0].Guid == "" {
		t.Fatalf("expected repo guid to be set")
	}
	if newConfig.Repos[0].AutoInitialize {
		t.Fatalf("expected repo auto-initialize to be false")
	}
}
