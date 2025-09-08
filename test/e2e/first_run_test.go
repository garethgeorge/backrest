package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/testutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestFirstRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "backrest-e2e-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDataToBackup := filepath.Join(tmpDir, "data-to-backup")
	if err := os.Mkdir(tmpDataToBackup, 0755); err != nil {
		t.Fatalf("failed to create temp data to backup dir: %v", err)
	}

	binPath := filepath.Join(tmpDir, "backrest")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Build backrest binary
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/backrest")
	buildCmd.Stderr = os.Stderr
	buildCmd.Stdout = os.Stdout
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build backrest binary: %v", err)
	}

	addr := testutil.AllocOpenBindAddr(t)

	// Run backrest
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath,
		"-data-dir", tmpDir, "-config-file",
		filepath.Join(tmpDir, "config.json"),
		"-bind-address", addr)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start backrest: %v", err)
	}

	testutil.TryNonfatal(t, ctx, func() error {
		resp, err := http.Get(fmt.Sprintf("http://%s", addr))
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		return nil
	})

	t.Run("set instance ID", func(t *testing.T) {
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)

		req := connect.NewRequest(&v1.Config{
			Instance: "TestInstance",
		})
		resp, err := client.SetConfig(context.Background(), req)
		if err != nil {
			t.Fatalf("SetConfig failed: %v", err)
		}

		if resp.Msg.Instance != "TestInstance" {
			t.Errorf("expected instance ID to be 'TestInstance', got %q", resp.Msg.Instance)
		}
	})

	t.Run("get config", func(t *testing.T) {
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)

		req := connect.NewRequest(&emptypb.Empty{})
		resp, err := client.GetConfig(context.Background(), req)
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}

		if resp.Msg.Instance != "TestInstance" {
			t.Errorf("expected instance ID to be 'TestInstance', got %q", resp.Msg.Instance)
		}
	})

	t.Run("add repo", func(t *testing.T) {
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)

		req := connect.NewRequest(&v1.Repo{
			Id:       "test-repo",
			Uri:      filepath.Join(tmpDir, "test-repo"),
			Password: "1234",
		})
		_, err := client.AddRepo(context.Background(), req)
		if err != nil {
			t.Fatalf("AddRepo failed: %v", err)
		}
	})

	t.Run("add plan", func(t *testing.T) {
		// flow is fetch config, add plan to plans field of config, update config
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)

		resp, err := client.GetConfig(context.Background(), connect.NewRequest(&emptypb.Empty{}))
		if err != nil {
			t.Fatalf("GetConfig failed: %v", err)
		}

		resp.Msg.Plans = append(resp.Msg.Plans, &v1.Plan{
			Id:   "test-plan",
			Repo: "test-repo",
			Paths: []string{
				tmpDataToBackup,
			},
		})

		_, err = client.SetConfig(context.Background(), connect.NewRequest(resp.Msg))
		if err != nil {
			t.Fatalf("SetConfig failed: %v", err)
		}
	})

	t.Run("run backup operation", func(t *testing.T) {
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)
		_, err := client.Backup(context.Background(), connect.NewRequest(&types.StringValue{
			Value: "test-plan",
		}))
		if err != nil {
			t.Fatalf("Backup failed: %v", err)
		}
	})

	for _, tc := range []struct {
		name string
		task v1.DoRepoTaskRequest_Task
	}{
		{
			name: "check",
			task: v1.DoRepoTaskRequest_TASK_CHECK,
		},
		{
			name: "prune",
			task: v1.DoRepoTaskRequest_TASK_PRUNE,
		},
		{
			name: "stats",
			task: v1.DoRepoTaskRequest_TASK_STATS,
		},
		{
			name: "index snapshots",
			task: v1.DoRepoTaskRequest_TASK_INDEX_SNAPSHOTS,
		},
	} {
		t.Run(fmt.Sprintf("trigger %s", tc.name), func(t *testing.T) {
			client := v1connect.NewBackrestClient(
				http.DefaultClient,
				fmt.Sprintf("http://%s", addr),
			)

			req := connect.NewRequest(&v1.DoRepoTaskRequest{
				RepoId: "test-repo",
				Task:   tc.task,
			})
			_, err := client.DoRepoTask(context.Background(), req)
			if err != nil {
				t.Fatalf("DoRepoTask failed: %v", err)
			}
		})
	}

	t.Run("get operations", func(t *testing.T) {
		client := v1connect.NewBackrestClient(
			http.DefaultClient,
			fmt.Sprintf("http://%s", addr),
		)

		req := connect.NewRequest(&v1.GetOperationsRequest{
			Selector: &v1.OpSelector{
				InstanceId: proto.String("TestInstance"),
			},
		})
		resp, err := client.GetOperations(context.Background(), req)
		if err != nil {
			t.Fatalf("GetOperations failed: %v", err)
		}

		if len(resp.Msg.Operations) == 0 {
			t.Errorf("expected at least 1 operation, got %d", len(resp.Msg.Operations))
		}
	})
}
