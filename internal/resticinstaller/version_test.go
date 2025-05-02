package resticinstaller

import "testing"

func TestParseSemVer(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    [3]int
		wantErr bool
	}{
		{"Valid version", "0.18.0", [3]int{0, 18, 0}, false},
		{"Invalid version", "1.2", [3]int{}, true},
		{"Empty string", "", [3]int{}, true},
		{"Non-numeric version", "a.b.c", [3]int{}, true},
		{"Version with extra parts", "1.2.3.4", [3]int{1, 2, 3}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSemVer(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseSemVer(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("parseSemVer(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestCompareSemVer(t *testing.T) {
	testCases := []struct {
		name string
		v1   [3]int
		v2   [3]int
		want int // 1 if v1 > v2, -1 if v1 < v2, 0 if v1 == v2
	}{
		{"Equal versions", [3]int{1, 2, 3}, [3]int{1, 2, 3}, 0},
		{"v1 major greater", [3]int{2, 0, 0}, [3]int{1, 9, 9}, 1},
		{"v1 major smaller", [3]int{1, 9, 9}, [3]int{2, 0, 0}, -1},
		{"v1 minor greater", [3]int{1, 3, 0}, [3]int{1, 2, 9}, 1},
		{"v1 minor smaller", [3]int{1, 2, 9}, [3]int{1, 3, 0}, -1},
		{"v1 patch greater", [3]int{1, 2, 4}, [3]int{1, 2, 3}, 1},
		{"v1 patch smaller", [3]int{1, 2, 3}, [3]int{1, 2, 4}, -1},
		{"Zero versions equal", [3]int{0, 0, 0}, [3]int{0, 0, 0}, 0},
		{"Mixed zero versions", [3]int{0, 1, 0}, [3]int{0, 0, 9}, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareSemVer(tc.v1, tc.v2)
			if got != tc.want {
				t.Errorf("compareSemVer(%v, %v) = %d, want %d", tc.v1, tc.v2, got, tc.want)
			}
		})
	}
}
