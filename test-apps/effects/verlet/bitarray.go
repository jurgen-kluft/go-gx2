package fx_verlet

type BitArray struct {
	Bits []uint32
	Size int32
}

func NewBitArray(size int32) BitArray {
	bitArray := BitArray{
		Size: size,
	}
	bitArray.Bits = make([]uint32, (size+31)/32)
	return bitArray
}

func (ba *BitArray) SetBit(index int32) {
	if index < 0 || index >= ba.Size {
		return
	}
	wordIndex := index >> 5 // Divide by 32
	bitIndex := index & 31
	ba.Bits[wordIndex] |= (1 << bitIndex)
}

func (ba *BitArray) ClearBit(index int32) {
	if index < 0 || index >= ba.Size {
		return
	}
	wordIndex := index >> 5 // Divide by 32
	bitIndex := index & 31
	ba.Bits[wordIndex] &^= (1 << bitIndex)
}

func (ba *BitArray) IsBitSet(index int32) bool {
	if index < 0 || index >= ba.Size {
		return false
	}
	wordIndex := index >> 5 // Divide by 32
	bitIndex := index & 31
	return (ba.Bits[wordIndex] & (1 << bitIndex)) != 0
}

// Iterate returns the index of the next set bit starting from the given index.
// User needs to start with index = -1 to find the first set bit.
// Returns false if no more set bits are found.
func (ba *BitArray) Next(index *int32) bool {
	i := *index
	if i < 0 {
		// Find the first set bit from the beginning
		for wi := int32(0); wi < int32(len(ba.Bits)); wi++ {
			word := ba.Bits[wi]
			if word != 0 {
				for bi := int32(0); bi < 32; bi++ {
					if (word & (1 << bi)) != 0 {
						*index = (wi << 5) + bi
						return true
					}
				}
			}
		}
	}

	i = *index + 1

	if i >= ba.Size {
		return false
	}

	// Do it smartly, check whole words first, then check the bits in the word.
	wordIndex := i >> 5
	bitIndex := i & 31

	// Check the current word
	if wordIndex < int32(len(ba.Bits)) {
		word := ba.Bits[wordIndex] &^ ((1 << bitIndex) - 1)
		if word != 0 {
			for i := bitIndex; i < 32; i++ {
				if (word & (1 << i)) != 0 {
					*index = (wordIndex << 5) + i
					return true
				}
			}
		}
	}

	// Check subsequent words
	for wi := wordIndex + 1; wi < int32(len(ba.Bits)); wi++ {
		word := ba.Bits[wi]
		if word != 0 {
			for i := int32(0); i < 32; i++ {
				if (word & (1 << i)) != 0 {
					*index = (wi << 5) + i
					return true
				}
			}
		}
	}

	return false
}
