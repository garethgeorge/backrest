package validationutil

import (
	"testing"
)

func TestSanitizeID(t *testing.T) {
	tcs := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "empty",
			id:   "",
			want: "",
		},
		{
			name: "no change",
			id:   "abc123",
			want: "abc123",
		},
		{
			name: "spaces",
			id:   "a b c 1 2 3",
			want: "a_b_c_1_2_3",
		},
		{
			name: "special characters",
			id:   "a!b@c#1$2%3",
			want: "a_b_c_1_2_3",
		},
		{
			name: "unicode",
			id:   "a👍b👍c👍1👍2👍3",
			want: "a_b_c_1_2_3",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeID(tc.id)
			if got != tc.want {
				t.Errorf("SanitizeID(%q) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
}
