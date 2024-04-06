//go:build windows
// +build windows

package restic

import "testing"

func TestToPathFilter(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		path string
		want string
	}{
		{"C:\\Users\\user\\Documents", "/C/Users/user/Documents"},
		{"C:\\", "/C"},
		{"C:", "/C"},
		{"/Users/user/Documents", "/Users/user/Documents"},
		{"\\Users\\user\\Documents", "/Users/user/Documents"},

		// network share - not sure if this is correct
		{"\\\\network-share\\path\\to\\file", "//network-share/path/to/file"},

		// invalid / not handled...
		{"1:\\foobar", "/1:/foobar"},
		{"AA:\\foobar", "/AA:/foobar"},
		{"a/relative/directory", "/a/relative/directory"},
	}

	for _, tc := range tcs {
		got := toPathFilter(tc.path)
		if got != tc.want {
			t.Errorf("toPathFilter(%q) == %q, want %q", tc.path, got, tc.want)
		}
	}
}
