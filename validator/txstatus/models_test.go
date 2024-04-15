package txstatus

import (
	"github.com/qubic/go-node-connector/types"
	"log"
	"testing"
)

func TestQubicToProto(t *testing.T) {
	tickTransactionStatus := types.TransactionStatus{
		CurrentTickOfNode: 100,
		Tick:              100,
		TxCount:           3,
		MoneyFlew:         [128]byte{0b10001000, 0b00000001},
		TransactionDigests: [][32]byte{
			[32]byte{209, 173, 239, 194, 151, 98, 29, 180, 83, 67, 142, 32, 4, 9, 167, 32, 159, 95, 116, 116, 214, 221, 171, 255, 13, 125, 86, 112, 5, 31, 191, 193},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 21},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},
			[32]byte{209, 173, 239, 194, 151, 98, 29, 180, 83, 67, 142, 32, 4, 9, 167, 32, 159, 95, 116, 116, 214, 221, 171, 255, 13, 125, 86, 112, 5, 31, 191, 193},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 21},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 21},
			[32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},
		},
	}

	res, err := qubicToProto(tickTransactionStatus, true)
	if err != nil {
		t.Fatalf("Got err when converting qubic to proto. err: %s", err.Error())
	}

	log.Println(res)
}

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
