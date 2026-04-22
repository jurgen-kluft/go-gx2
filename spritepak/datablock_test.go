package spritepak

import (
	"reflect"
	"testing"
)

// TestReuseDataBlocks tests the reuseDataBlocks function to ensure it correctly identifies and reuses identical data blocks.
func TestReuseDataBlocks(t *testing.T) {
	// Test case with some identical data blocks
	dataBlocks := [][]uint16{
		{1, 2, 3},
		{4, 5, 6},
		{1, 2, 3}, // identical to the first block
		{7, 8, 9},
		{4, 5, 6}, // identical to the second block
	}

	refArray, dataArray := reuseDataBlocks(dataBlocks)

	// Expected results
	expectedRefArray := []int{0, 1, 0, 2, 1}
	expectedDataArray := [][]uint16{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	// Check if the reference array matches the expected reference array
	if !reflect.DeepEqual(refArray, expectedRefArray) {
		t.Errorf("Expected refArray %v but got %v", expectedRefArray, refArray)
	}

	// Check if the data array matches the expected data array
	if !reflect.DeepEqual(dataArray, expectedDataArray) {
		t.Errorf("Expected dataArray %v but got %v", expectedDataArray, dataArray)
	}
}
