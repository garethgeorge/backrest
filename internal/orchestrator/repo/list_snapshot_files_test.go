package repo

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestNormalizeSnapshotListPath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":          "/",
		"/":         "/",
		"foo":       "/foo",
		"/foo":      "/foo",
		"/foo/":     "/foo",
		"/foo/bar/": "/foo/bar",
	}

	for input, want := range cases {
		if got := normalizeSnapshotListPath(input); got != want {
			t.Errorf("normalizeSnapshotListPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDirectSnapshotChildren(t *testing.T) {
	t.Parallel()

	entries := []*v1.LsEntry{
		{Path: "/", Type: "directory"},
		{Path: "/one", Type: "directory"},
		{Path: "/one/nested", Type: "directory"},
		{Path: "/two", Type: "file"},
	}

	children := directSnapshotChildren("/", entries)
	if len(children) != 2 {
		t.Fatalf("directSnapshotChildren returned %d entries, want 2", len(children))
	}
	if children[0].Path != "/one" || children[1].Path != "/two" {
		t.Fatalf("directSnapshotChildren returned paths %q and %q, want /one and /two", children[0].Path, children[1].Path)
	}

	children = directSnapshotChildren("/one/", entries)
	if len(children) != 1 || children[0].Path != "/one/nested" {
		t.Fatalf("directSnapshotChildren for /one returned %#v, want only /one/nested", children)
	}
}
