package fx_verlet

import "testing"

func collectSetBits(t *testing.T, bitArray *BitArray, start int32) []int32 {
	index := start
	collected := make([]int32, 0)
	for bitArray.Next(&index) {
		collected = append(collected, index)
	}
	return collected
}

func assertBitSequence(t *testing.T, actual []int32, expected []int32) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %d bits, got %d: %v", len(expected), len(actual), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected bit sequence %v, got %v", expected, actual)
		}
	}
}

func TestNewBitArrayAllocatesExpectedWords(t *testing.T) {
	tests := []struct {
		name          string
		size          int32
		expectedWords int
	}{
		{name: "zero size", size: 0, expectedWords: 0},
		{name: "single bit", size: 1, expectedWords: 1},
		{name: "partial word", size: 31, expectedWords: 1},
		{name: "full word", size: 32, expectedWords: 1},
		{name: "word plus one", size: 33, expectedWords: 2},
		{name: "two full words", size: 64, expectedWords: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bitArray := NewBitArray(test.size)
			if bitArray.Size != test.size {
				t.Fatalf("expected size %d, got %d", test.size, bitArray.Size)
			}
			if len(bitArray.Bits) != test.expectedWords {
				t.Fatalf("expected %d words, got %d", test.expectedWords, len(bitArray.Bits))
			}
		})
	}
}

func TestBitArraySetClearAndQueryAcrossWordBoundaries(t *testing.T) {
	bitArray := NewBitArray(65)
	targets := []int32{0, 31, 32, 33, 64}
	neighbors := []int32{1, 30, 34, 63}

	for _, index := range targets {
		if bitArray.IsBitSet(index) {
			t.Fatalf("bit %d should be clear initially", index)
		}
		bitArray.SetBit(index)
		if !bitArray.IsBitSet(index) {
			t.Fatalf("bit %d should be set", index)
		}
		bitArray.SetBit(index)
		if !bitArray.IsBitSet(index) {
			t.Fatalf("bit %d should remain set after setting twice", index)
		}
	}

	for _, index := range neighbors {
		if bitArray.IsBitSet(index) {
			t.Fatalf("neighbor bit %d should remain clear", index)
		}
	}

	for _, index := range targets {
		bitArray.ClearBit(index)
		if bitArray.IsBitSet(index) {
			t.Fatalf("bit %d should be clear after ClearBit", index)
		}
		bitArray.ClearBit(index)
		if bitArray.IsBitSet(index) {
			t.Fatalf("bit %d should remain clear after clearing twice", index)
		}
	}
}

func TestBitArrayOutOfRangeOperationsAreNoOps(t *testing.T) {
	bitArray := NewBitArray(65)
	bitArray.SetBit(32)

	for _, index := range []int32{-1, 65, 66} {
		bitArray.SetBit(index)
		bitArray.ClearBit(index)
		if bitArray.IsBitSet(index) {
			t.Fatalf("out-of-range bit %d should always query as clear", index)
		}
	}

	if !bitArray.IsBitSet(32) {
		t.Fatalf("valid bit was unexpectedly modified by out-of-range operations")
	}

	empty := NewBitArray(0)
	for _, index := range []int32{-1, 0, 1} {
		empty.SetBit(index)
		empty.ClearBit(index)
		if empty.IsBitSet(index) {
			t.Fatalf("zero-sized array should report bit %d as clear", index)
		}
	}
}

func TestBitArrayNextFindsBitsInOrder(t *testing.T) {
	tests := []struct {
		name     string
		size     int32
		setBits  []int32
		expected []int32
	}{
		{name: "empty array", size: 0, expected: []int32{}},
		{name: "all clear", size: 96, expected: []int32{}},
		{name: "first bit at zero", size: 8, setBits: []int32{0}, expected: []int32{0}},
		{name: "consecutive bits", size: 16, setBits: []int32{4, 5, 6}, expected: []int32{4, 5, 6}},
		{name: "sparse bits", size: 96, setBits: []int32{2, 5, 70}, expected: []int32{2, 5, 70}},
		{name: "word boundaries", size: 96, setBits: []int32{31, 32, 63, 64}, expected: []int32{31, 32, 63, 64}},
		{name: "empty words between bits", size: 130, setBits: []int32{0, 33, 66, 129}, expected: []int32{0, 33, 66, 129}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bitArray := NewBitArray(test.size)
			for _, index := range test.setBits {
				bitArray.SetBit(index)
			}

			actual := collectSetBits(t, &bitArray, -1)
			assertBitSequence(t, actual, test.expected)
		})
	}
}

func TestBitArrayNextResumesAndStopsCorrectly(t *testing.T) {
	bitArray := NewBitArray(65)
	for _, index := range []int32{0, 31, 32, 64} {
		bitArray.SetBit(index)
	}

	resumed := collectSetBits(t, &bitArray, 31)
	assertBitSequence(t, resumed, []int32{32, 64})

	index := int32(64)
	if bitArray.Next(&index) {
		t.Fatalf("expected no next bit after the last set bit")
	}
	if index != 64 {
		t.Fatalf("expected index to remain unchanged after exhaustion, got %d", index)
	}

	missing := int32(-1)
	clearArray := NewBitArray(32)
	if clearArray.Next(&missing) {
		t.Fatalf("expected empty bit array iteration to fail")
	}
	if missing != -1 {
		t.Fatalf("expected index to remain unchanged when no set bit is found, got %d", missing)
	}
}

func TestBitArrayNextSkipsClearedBits(t *testing.T) {
	bitArray := NewBitArray(16)
	for _, index := range []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		bitArray.SetBit(index)
	}
	bitArray.ClearBit(5)

	actual := collectSetBits(t, &bitArray, -1)
	assertBitSequence(t, actual, []int32{0, 1, 2, 3, 4, 6, 7, 8, 9, 10})
}
