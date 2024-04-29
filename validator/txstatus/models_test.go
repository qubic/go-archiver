package txstatus

import (
	"testing"
)

func TestEqualSlices(t *testing.T) {
	slice1 := [][32]byte{
		[32]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
		[32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		[32]byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	}

	slice2 := [][32]byte{
		[32]byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		[32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		[32]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
	}

	if !equalDigests(slice1, slice2) {
		t.Fatalf("Slices are not equal")
	}

	// check if the original slices were not changed
	if slice1[0] != [32]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30} {
		t.Fatalf("Original slice was changed")
	}
	if slice2[0] != [32]byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 20} {
		t.Fatalf("Original slice was changed")
	}
	if slice1[1] != [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		t.Fatalf("Original slice was changed")
	}
	if slice2[1] != [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		t.Fatalf("Original slice was changed")
	}
	if slice1[2] != [32]byte{11, 12, 13, 14, 15, 16, 17, 18, 19, 20} {
		t.Fatalf("Original slice was changed")
	}
	if slice2[2] != [32]byte{21, 22, 23, 24, 25, 26, 27, 28, 29, 30} {
		t.Fatalf("Original slice was changed")
	}

}
