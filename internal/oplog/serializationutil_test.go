package oplog

import "testing"

func TestItoa(t *testing.T) {
	nums := []int64{0, 1, 2, 3, 4, 1 << 32, int64(1) << 62}
	for _, num := range nums {
		b := itob(num)
		if btoi(b) != num {
			t.Errorf("itob/btoi failed for %d", num)
		}
	}
}