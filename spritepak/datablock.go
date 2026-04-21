package spritepak

import (
	"crypto/sha1"
	"encoding/binary"
)

// DataElement constrains the element types supported by reuseDataBlocks.
type DataElement interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | float32 | float64
}

// reuseDataBlocks identifies identical data blocks and reuses them to save space.
func reuseDataBlocks[T DataElement](dataBlocks [][]T) ([]int, [][]T) {
	// Identify data blocks that are identical and reuse
	hasher := sha1.New()
	dataArray := make([][]T, 0, len(dataBlocks))
	refArray := make([]int, 0, len(dataBlocks))
	dataMap := make(map[string]int)
	for _, pd := range dataBlocks {
		if pd == nil {
			continue
		}

		// SHA1 hash of data used to identify identical data blocks
		hasher.Reset()
		for _, c := range pd {
			binary.Write(hasher, binary.LittleEndian, c)
		}
		hash := string(hasher.Sum(nil))

		if ref, found := dataMap[hash]; found {
			// Reuse existing data block
			refArray = append(refArray, ref)
		} else {
			// Store new data block
			ref = len(dataArray)
			dataMap[hash] = ref
			refArray = append(refArray, ref)
			dataArray = append(dataArray, pd)
		}
	}
	return refArray, dataArray
}
