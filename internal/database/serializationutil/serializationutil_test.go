package serializationutil

import "testing"

func TestItoa(t *testing.T) {
	nums := []int64{0, 1, 2, 3, 4, 1 << 32, int64(1) << 62}
	for _, num := range nums {
		b := Itob(num)
		if Btoi(b) != num {
			t.Errorf("itob/btoi failed for %d", num)
		}
	}
}

func TestStob(t *testing.T) {
	strs := []string{"", "a", "ab", "abc", "abcd", "abcde", "abcdef"}
	for _, str := range strs {
		b := Stob(str)
		if val, _ := Btos(b); val != str {
			t.Errorf("stob/btos failed for %s", str)
		}
	}
}
