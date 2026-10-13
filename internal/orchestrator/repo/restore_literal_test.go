package repo

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestEscapeGlobLiteralBrackets(t *testing.T) {
	for _, name := range []string{
		"plain.txt", "[", "]", "[]", "][", "[[]", "[]]", "[[[]]]",
		"open[.txt", "close].txt", "report[1].txt", "[-].txt", "[^a-z].txt",
		"[!].txt", "日本語[é].txt",
	} {
		t.Run(name, func(t *testing.T) {
			pattern := escapeGlob(name)
			// Restic also checks the pattern's syntax by matching it against itself.
			if _, err := filepath.Match(pattern, pattern); err != nil {
				t.Fatalf("invalid include pattern %q: %v", pattern, err)
			}
			if match, err := filepath.Match(pattern, name); err != nil || !match {
				t.Fatalf("%q must match %q: match=%v, err=%v", pattern, name, match, err)
			}
			for i, character := range name {
				if character != '[' && character != ']' {
					continue
				}
				for _, replacement := range []string{"X", "é", "[", "]"} {
					sibling := name[:i] + replacement + name[i+1:]
					if sibling == name {
						continue
					}
					if match, err := filepath.Match(pattern, sibling); err != nil || match {
						t.Errorf("%q must not match sibling %q: match=%v, err=%v", pattern, sibling, match, err)
					}
				}
			}
		})
	}
}

// Exercise actual restic filtering, including its validation of the generated
// include pattern. The destination deliberately contains users' existing files:
// restoring an extra sibling and deleting it afterward would fail this test too.
func TestRestoreLiteralBrackets(t *testing.T) {
	t.Parallel()
	source := t.TempDir()
	wantedTime := time.Date(2020, 4, 5, 6, 7, 8, 0, time.UTC)
	write := func(name, content string) {
		t.Helper()
		filename := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filename, wantedTime, wantedTime); err != nil {
			t.Fatal(err)
		}
	}
	wanted := []string{"report[1].txt", "open[.txt", "close].txt", "[-].txt", "[^a-z].txt", "日本語[é].txt"}
	for _, name := range wanted {
		write(name, "selected "+name)
		write(strings.NewReplacer("[", "X", "]", "Y").Replace(name), "unselected sibling")
		write(strings.NewReplacer("[", "A", "]", "B").Replace(name), "unselected missing sibling")
	}
	write("folder[1]/nested/selected.txt", "selected child")
	write("folderX1Y/nested/unselected.txt", "unselected directory")
	write("folderA1B/nested/unselected.txt", "unselected missing directory")
	if err := os.MkdirAll(filepath.Join(source, "folder[1]", "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "empty[1]"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "emptyX1Y"), 0755); err != nil {
		t.Fatal(err)
	}

	r := &v1.Repo{Id: "literal-brackets", Uri: t.TempDir(), Password: "test", Flags: []string{"--no-cache"}}
	orchestrator := initRepoHelper(t, configForTest, r)
	backup, err := orchestrator.Backup(context.Background(), &v1.Plan{Id: "test", Repo: r.Id, Paths: []string{source}}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshotPath := filepath.ToSlash(source)
	if runtime.GOOS == "windows" {
		snapshotPath = "/" + strings.ReplaceAll(snapshotPath, ":", "")
	}

	for _, name := range append(wanted, "folder[1]", "empty[1]") {
		t.Run(name, func(t *testing.T) {
			target := t.TempDir()
			sibling := strings.NewReplacer("[", "X", "]", "Y").Replace(name)
			// For the file cases, a broad match would overwrite this sentinel.
			// For directory cases, it would replace this file with a directory.
			sentinel := filepath.Join(target, sibling)
			if err := os.WriteFile(sentinel, []byte("user's existing file"), 0600); err != nil {
				t.Fatal(err)
			}
			var finalProgress *v1.RestoreProgressEntry
			summary, err := orchestrator.Restore(context.Background(), backup.SnapshotId, path.Join(snapshotPath, name), target, func(event *v1.RestoreProgressEntry) {
				finalProgress = event
			})
			if err != nil {
				t.Fatal(err)
			}
			if finalProgress == nil || finalProgress.MessageType != "summary" || finalProgress.FilesRestored != summary.FilesRestored {
				t.Fatalf("final progress does not agree with summary: progress=%v summary=%v", finalProgress, summary)
			}
			if content, err := os.ReadFile(sentinel); err != nil || string(content) != "user's existing file" {
				t.Fatalf("unselected existing file changed: content=%q err=%v", content, err)
			}
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 2 {
				t.Fatalf("expected only selected path and user's file: entries=%v err=%v", entries, err)
			}
			restored := filepath.Join(target, name)
			if name == "folder[1]" {
				content, err := os.ReadFile(filepath.Join(restored, "nested", "selected.txt"))
				if err != nil || string(content) != "selected child" {
					t.Fatalf("selected directory was not restored: content=%q err=%v", content, err)
				}
				if info, err := os.Stat(filepath.Join(restored, "empty")); err != nil || !info.IsDir() {
					t.Fatalf("empty child directory was not restored: %v", err)
				}
			} else if name == "empty[1]" {
				if info, err := os.Stat(restored); err != nil || !info.IsDir() {
					t.Fatalf("selected empty directory was not restored: %v", err)
				}
			} else {
				content, err := os.ReadFile(restored)
				if err != nil || string(content) != "selected "+name {
					t.Fatalf("selected file was not restored: content=%q err=%v", content, err)
				}
				info, err := os.Stat(restored)
				if err != nil || !info.ModTime().Equal(wantedTime) {
					t.Fatalf("file timestamp was not preserved: info=%v err=%v", info, err)
				}
				if summary.TotalFiles != 1 || summary.FilesRestored != 1 {
					t.Fatalf("expected exactly one restored file: %v", summary)
				}
			}
		})
	}

	t.Run("cancelled restore", func(t *testing.T) {
		target := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := orchestrator.Restore(ctx, backup.SnapshotId, path.Join(snapshotPath, wanted[0]), target, nil); err == nil {
			t.Fatal("expected cancellation error")
		}
		if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
			t.Fatalf("cancelled restore wrote to the destination: entries=%v err=%v", entries, err)
		}
	})
}
