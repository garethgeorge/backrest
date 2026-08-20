package repo

import "testing"

func TestStripEnvValueQuotes(t *testing.T) {
	tcs := []struct {
		name string
		in   string
		want string
	}{
		{"no quotes", "KEY=value", "KEY=value"},
		{"double quotes", `KEY="value"`, "KEY=value"},
		{"single quotes", "KEY='value'", "KEY=value"},
		{"empty double quotes", `KEY=""`, "KEY="},
		{"empty single quotes", "KEY=''", "KEY="},
		{"no value", "KEY=", "KEY="},
		{"mismatched quotes left untouched", `KEY="value'`, `KEY="value'`},
		{"quote only inside value untouched", `KEY=va"lu"e`, `KEY=va"lu"e`},
		{"no equals sign untouched", "NOEQUALSSIGN", "NOEQUALSSIGN"},
		{"value with embedded equals", `KEY="a=b"`, "KEY=a=b"},
		{"double quotes around base64-like value", `AZURE_ACCOUNT_KEY="abc123=="`, "AZURE_ACCOUNT_KEY=abc123=="},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := StripEnvValueQuotes(tc.in)
			if got != tc.want {
				t.Errorf("StripEnvValueQuotes(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
