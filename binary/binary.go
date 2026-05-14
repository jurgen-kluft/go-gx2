// Package binary_format provides binary encoding/decoding helpers including
// automatic reflection-based struct serialization with deferred pointer support.
package binary_format

import (
	"bytes"
	"crypto/sha1"
	stdbinary "encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"unicode/utf8"
	"unsafe"
)

// Writer is a buffered binary writer with deferred pointer support.
type Writer struct {
	buf      *bytes.Buffer
	order    stdbinary.ByteOrder
	offset   int64
	deferred *DeferredDataBlocks
	filter   FieldFilter
	temp     [8]byte
}

// NewWriter constructs a buffered binary writer using the provided byte order.
func NewWriter(order stdbinary.ByteOrder) *Writer {
	if order == nil {
		order = stdbinary.LittleEndian
	}
	return &Writer{
		buf:      &bytes.Buffer{},
		order:    order,
		deferred: NewDeferredDataBlocks(),
	}
}

// Order returns the byte order used by the writer.
func (w *Writer) Order() stdbinary.ByteOrder {
	return w.order
}

// Offset returns the number of bytes written so far.
func (w *Writer) Offset() int64 {
	return w.offset
}

// Buffer returns the underlying bytes.Buffer.
func (w *Writer) Buffer() *bytes.Buffer {
	return w.buf
}

// Deferred returns the deferred data block manager.
func (w *Writer) Deferred() *DeferredDataBlocks {
	return w.deferred
}

// WriteBytes writes the supplied bytes as-is.
func (w *Writer) WriteBytes(value []byte) error {
	if len(value) == 0 {
		return nil
	}
	n, err := w.buf.Write(value)
	w.offset += int64(n)
	if err != nil {
		return err
	}
	if n != len(value) {
		return io.ErrShortWrite
	}
	return nil
}

// WriteUint8 writes a uint8 value.
func (w *Writer) WriteUint8(value uint8) error {
	w.temp[0] = value
	return w.WriteBytes(w.temp[:1])
}

// WriteInt8 writes an int8 value.
func (w *Writer) WriteInt8(value int8) error {
	return w.WriteUint8(uint8(value))
}

// WriteUint16 writes a uint16 value.
func (w *Writer) WriteUint16(value uint16) error {
	w.order.PutUint16(w.temp[:2], value)
	return w.WriteBytes(w.temp[:2])
}

// WriteInt16 writes an int16 value.
func (w *Writer) WriteInt16(value int16) error {
	return w.WriteUint16(uint16(value))
}

// WriteUint32 writes a uint32 value.
func (w *Writer) WriteUint32(value uint32) error {
	w.order.PutUint32(w.temp[:4], value)
	return w.WriteBytes(w.temp[:4])
}

// WriteInt32 writes an int32 value.
func (w *Writer) WriteInt32(value int32) error {
	return w.WriteUint32(uint32(value))
}

// WriteUint64 writes a uint64 value.
func (w *Writer) WriteUint64(value uint64) error {
	w.order.PutUint64(w.temp[:8], value)
	return w.WriteBytes(w.temp[:8])
}

// WriteInt64 writes an int64 value.
func (w *Writer) WriteInt64(value int64) error {
	return w.WriteUint64(uint64(value))
}

// WriteFloat32 writes a float32 value.
func (w *Writer) WriteFloat32(value float32) error {
	return w.WriteUint32(math.Float32bits(value))
}

// WriteFloat64 writes a float64 value.
func (w *Writer) WriteFloat64(value float64) error {
	return w.WriteUint64(math.Float64bits(value))
}

// Bytes returns a copy of the buffer contents.
func (w *Writer) Bytes() []byte {
	return w.buf.Bytes()
}

// WriteDeferredData writes all deferred data blocks sequentially and updates all tracked pointers.
// Only writes canonical blocks (deduplicates by SHA1), so identical data is not repeated.
// Call this after writing all structures.
func (w *Writer) WriteDeferredData() error {
	result := w.buf.Bytes()
	currentOffset := uint64(len(result))

	// Map canonical block index to its actual offset in output
	blockOffsets := make([]uint64, len(w.deferred.blocks))

	// Write only canonical blocks (DedupIndex == -1)
	for i, block := range w.deferred.blocks {
		if block.DedupIndex == -1 {
			alignment := normalizeBlockAlignment(block.Alignment)
			if alignment > 1 {
				padding := (uint64(alignment) - (currentOffset % uint64(alignment))) % uint64(alignment)
				if padding > 0 {
					pad := make([]byte, int(padding))
					if err := w.WriteBytes(pad); err != nil {
						return err
					}
					currentOffset += padding
				}
			}

			// This is a canonical block, write it
			blockOffsets[i] = currentOffset
			dataToWrite := block.Data
			if len(block.Relocations) > 0 {
				dataToWrite = make([]byte, len(block.Data))
				copy(dataToWrite, block.Data)
				for _, rel := range block.Relocations {
					if rel < 0 || rel+8 > int64(len(dataToWrite)) {
						return fmt.Errorf("binary_format: relocation offset out of bounds: %d", rel)
					}
					ptr := w.order.Uint64(dataToWrite[rel : rel+8])
					w.order.PutUint64(dataToWrite[rel:rel+8], ptr+currentOffset)
				}
			}

			if err := w.WriteBytes(dataToWrite); err != nil {
				return err
			}
			currentOffset += uint64(len(block.Data))
		}
		// For deduplicated blocks, blockOffsets[i] will be set below
	}

	// Update all tracked pointers with actual data offsets
	result = w.buf.Bytes() // refresh after writes
	for blockIndex, block := range w.deferred.blocks {
		// Determine which canonical block to point to
		offsetToUse := blockOffsets[blockIndex]

		for _, ptrLoc := range block.PointerLocations {
			w.order.PutUint64(result[ptrLoc:ptrLoc+8], offsetToUse)
		}
	}

	return nil
}

// WriteStringPointer writes a C-style deferred string to this writer.
// Format: [u64 pointer, u32 byte_count, u32 char_count] with UTF-8 data deferred for later.
func (w *Writer) WriteStringPointer(value string) error {
	byteData := []byte(value)
	charCount := utf8.RuneCountInString(value)

	// Record pointer location and add deferred data block.
	ptrLoc := w.Offset()
	blockIndex := w.deferred.AddDataBlockAligned(byteData, 1)
	w.deferred.AddPointerLocation(blockIndex, ptrLoc)

	// Write placeholder pointer.
	if err := w.WriteUint64(0); err != nil {
		return err
	}
	// Write byte count.
	if err := w.WriteUint32(uint32(len(byteData))); err != nil {
		return err
	}
	// Write character count.
	if err := w.WriteUint32(uint32(charCount)); err != nil {
		return err
	}

	return nil
}

// Reader is a buffered binary reader for decoding from bytes.
type Reader struct {
	data   []byte
	order  stdbinary.ByteOrder
	base   int64
	offset int64
	filter FieldFilter
	temp   [8]byte
}

// FieldFilter decides whether a struct field should be serialized.
// Returning true includes the field; false excludes it.
// Field matching is by struct declaration metadata (including field name).
type FieldFilter func(structType reflect.Type, field reflect.StructField) bool

// NewReader constructs a buffered binary reader from bytes using the provided byte order.
func NewReader(data []byte, order stdbinary.ByteOrder) *Reader {
	if order == nil {
		order = stdbinary.LittleEndian
	}
	return &Reader{data: data, order: order, base: 0}
}

// SetFieldFilter registers a field filter for automatic reflection-based serialization.
func (w *Writer) SetFieldFilter(filter FieldFilter) {
	w.filter = filter
}

// SetFieldFilter registers a field filter for automatic reflection-based deserialization.
func (r *Reader) SetFieldFilter(filter FieldFilter) {
	r.filter = filter
}

// Order returns the byte order used by the reader.
func (r *Reader) Order() stdbinary.ByteOrder {
	return r.order
}

// Offset returns the number of bytes read so far.
func (r *Reader) Offset() int64 {
	return r.offset
}

// ReadBytes reads exactly len(dest) bytes from the buffer into dest.
func (r *Reader) ReadBytes(dest []byte) error {
	if len(dest) == 0 {
		return nil
	}
	if r.offset+int64(len(dest)) > int64(len(r.data)) {
		return io.EOF
	}
	copy(dest, r.data[r.offset:r.offset+int64(len(dest))])
	r.offset += int64(len(dest))
	return nil
}

// Uint8 reads a uint8 value.
func (r *Reader) Uint8() (uint8, error) {
	if err := r.ReadBytes(r.temp[:1]); err != nil {
		return 0, err
	}
	return r.temp[0], nil
}

// Int8 reads an int8 value.
func (r *Reader) Int8() (int8, error) {
	v, err := r.Uint8()
	return int8(v), err
}

// Uint16 reads a uint16 value.
func (r *Reader) Uint16() (uint16, error) {
	if err := r.ReadBytes(r.temp[:2]); err != nil {
		return 0, err
	}
	return r.order.Uint16(r.temp[:2]), nil
}

// Int16 reads an int16 value.
func (r *Reader) Int16() (int16, error) {
	v, err := r.Uint16()
	return int16(v), err
}

// Uint32 reads a uint32 value.
func (r *Reader) Uint32() (uint32, error) {
	if err := r.ReadBytes(r.temp[:4]); err != nil {
		return 0, err
	}
	return r.order.Uint32(r.temp[:4]), nil
}

// Int32 reads an int32 value.
func (r *Reader) Int32() (int32, error) {
	v, err := r.Uint32()
	return int32(v), err
}

// Uint64 reads a uint64 value.
func (r *Reader) Uint64() (uint64, error) {
	if err := r.ReadBytes(r.temp[:8]); err != nil {
		return 0, err
	}
	return r.order.Uint64(r.temp[:8]), nil
}

// Int64 reads an int64 value.
func (r *Reader) Int64() (int64, error) {
	v, err := r.Uint64()
	return int64(v), err
}

// Float32 reads a float32 value.
func (r *Reader) Float32() (float32, error) {
	v, err := r.Uint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

// Float64 reads a float64 value.
func (r *Reader) Float64() (float64, error) {
	v, err := r.Uint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

// DeferredDataBlock represents a single deferred data block with its pointer locations.
type DeferredDataBlock struct {
	Data             []byte
	Hash             [20]byte // SHA1 hash of data for deduplication
	PointerLocations []int64  // offsets in the main buffer where pointers to this data are written
	Relocations      []int64  // offsets inside Data containing relative pointer values
	Alignment        int      // required alignment for the start of this block (power of two)
	DedupIndex       int      // if deduplicated, points to the canonical block index; -1 if canonical
}

// DeferredDataBlocks manages pointer placeholders and deferred data blocks.
// Deduplicates blocks by SHA1 hash to avoid writing identical data multiple times.
type DeferredDataBlocks struct {
	blocks  []DeferredDataBlock
	hashMap map[[20]byte]int // maps SHA1 hash to canonical block index
}

// NewDeferredDataBlocks creates a new deferred data block manager.
func NewDeferredDataBlocks() *DeferredDataBlocks {
	return &DeferredDataBlocks{
		blocks:  make([]DeferredDataBlock, 0),
		hashMap: make(map[[20]byte]int),
	}
}

func normalizeBlockAlignment(alignment int) int {
	if alignment <= 0 {
		return 1
	}
	if alignment > 8 {
		alignment = 8
	}
	if alignment&(alignment-1) != 0 {
		a := 1
		for a < alignment {
			a <<= 1
		}
		alignment = a
		if alignment > 8 {
			alignment = 8
		}
	}
	return alignment
}

// AddDataBlock records data to be written later and returns the index.
// The caller must track where the pointer was written using AddPointe (canonical or deduplicated).
// Automatically deduplicates by SHA1 hash - if identical data was added before,
// returns the index of the existing block instead of creating a new one.
func (ddb *DeferredDataBlocks) AddDataBlock(data []byte) int {
	return ddb.AddDataBlockAligned(data, 1)
}

// AddDataBlockAligned records data to be written later with a requested block-start alignment.
func (ddb *DeferredDataBlocks) AddDataBlockAligned(data []byte, alignment int) int {
	alignment = normalizeBlockAlignment(alignment)

	// Compute SHA1 hash of the data
	hash := sha1.Sum(data)

	// Check if we've already seen this exact data
	if canonicalIndex, exists := ddb.hashMap[hash]; exists {
		if alignment > ddb.blocks[canonicalIndex].Alignment {
			ddb.blocks[canonicalIndex].Alignment = alignment
		}
		// Data already exists, return canonical index
		return canonicalIndex
	}

	// New unique data block
	index := len(ddb.blocks)
	ddb.blocks = append(ddb.blocks, DeferredDataBlock{
		Data:             data,
		Hash:             hash,
		PointerLocations: make([]int64, 0),
		Relocations:      nil,
		Alignment:        alignment,
		DedupIndex:       -1, // This is a canonical block
	})
	ddb.hashMap[hash] = index
	return index
}

// AddDataBlockWithRelocations records a block whose embedded pointer values are relative to
// the start of the block and need to be relocated when the block is emitted.
// This method intentionally skips deduplication.
func (ddb *DeferredDataBlocks) AddDataBlockWithRelocations(data []byte, relocations []int64) int {
	return ddb.AddDataBlockWithRelocationsAligned(data, relocations, 1)
}

// AddDataBlockWithRelocationsAligned records a relocatable block with explicit block-start alignment.
// This method intentionally skips deduplication.
func (ddb *DeferredDataBlocks) AddDataBlockWithRelocationsAligned(data []byte, relocations []int64, alignment int) int {
	alignment = normalizeBlockAlignment(alignment)

	relocCopy := make([]int64, len(relocations))
	copy(relocCopy, relocations)

	index := len(ddb.blocks)
	ddb.blocks = append(ddb.blocks, DeferredDataBlock{
		Data:             data,
		PointerLocations: make([]int64, 0),
		Relocations:      relocCopy,
		Alignment:        alignment,
		DedupIndex:       -1,
	})
	return index
}

// AddPointerLocation records a location where a pointer to a data block was written.
func (ddb *DeferredDataBlocks) AddPointerLocation(blockIndex int, pointerLocation int64) {
	if blockIndex < 0 || blockIndex >= len(ddb.blocks) {
		return
	}
	ddb.blocks[blockIndex].PointerLocations = append(ddb.blocks[blockIndex].PointerLocations, pointerLocation)
}

// Blocks returns the managed data blocks.
func (ddb *DeferredDataBlocks) Blocks() []DeferredDataBlock {
	return ddb.blocks
}

// AlignTo writes padding bytes to align the current offset to the specified boundary.
// Returns the number of padding bytes written.
func AlignTo(writer *Writer, boundary int64) (int64, error) {
	if boundary <= 0 {
		return 0, nil
	}
	current := writer.Offset()
	padding := (boundary - (current % boundary)) % boundary
	if padding > 0 {
		pad := make([]byte, padding)
		if err := writer.WriteBytes(pad); err != nil {
			return 0, err
		}
	}
	return padding, nil
}

// ReadStringPointer reads a deferred UTF-8 string written as
// [u64 pointer, u32 byte_count, u32 char_count].
func (r *Reader) ReadStringPointer() (string, error) {
	ptr, err := r.Uint64()
	if err != nil {
		return "", err
	}
	byteCount, err := r.Uint32()
	if err != nil {
		return "", err
	}
	_, err = r.Uint32() // char_count, kept for format compatibility
	if err != nil {
		return "", err
	}

	if ptr >= uint64(len(r.data)) {
		return "", fmt.Errorf("binary_format: invalid string pointer offset %d", ptr)
	}
	if ptr+uint64(byteCount) > uint64(len(r.data)) {
		return "", fmt.Errorf("binary_format: string data extends past buffer")
	}

	return string(r.data[ptr : ptr+uint64(byteCount)]), nil
}

// WriteAuto serializes a struct value using reflection in declaration order with padding.
// Supported field kinds include numeric scalars, string, struct, pointer, slice, array, and map.
func (w *Writer) WriteAuto(value interface{}) error {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return fmt.Errorf("binary_format: WriteAuto requires a non-nil value")
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("binary_format: WriteAuto cannot serialize a nil pointer root")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("binary_format: WriteAuto root must be a struct or *struct, got %s", v.Kind())
	}
	return w.writeAutoStruct(v)
}

// ReadAuto deserializes into a struct pointer using reflection in declaration order with padding.
func (r *Reader) ReadAuto(dest interface{}) error {
	v := reflect.ValueOf(dest)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("binary_format: ReadAuto requires a non-nil pointer to struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("binary_format: ReadAuto destination must be *struct, got %s", v.Kind())
	}
	return r.readAutoStruct(v)
}

func (w *Writer) includeField(structType reflect.Type, field reflect.StructField) bool {
	if w.filter == nil {
		return true
	}
	return w.filter(structType, field)
}

func (r *Reader) includeField(structType reflect.Type, field reflect.StructField) bool {
	if r.filter == nil {
		return true
	}
	return r.filter(structType, field)
}

func (w *Writer) writeAutoStruct(value reflect.Value) error {
	t := value.Type()
	structAlign := autoTypeAlign(t)
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		if !w.includeField(t, field) {
			continue
		}
		if _, err := AlignTo(w, int64(autoTypeAlign(field.Type))); err != nil {
			return err
		}
		if err := w.writeAutoValue(value.Field(i)); err != nil {
			return fmt.Errorf("binary_format: write field %s: %w", field.Name, err)
		}
	}
	_, err := AlignTo(w, int64(structAlign))
	return err
}

func (r *Reader) readAutoStruct(value reflect.Value) error {
	t := value.Type()
	structAlign := autoTypeAlign(t)
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		if !r.includeField(t, field) {
			continue
		}
		if err := r.skipAutoPadding(int64(autoTypeAlign(field.Type))); err != nil {
			return err
		}
		fieldVal := value.Field(i)
		if !fieldVal.CanSet() {
			// Unexported field: obtain a settable reference via unsafe.
			fieldVal = reflect.NewAt(field.Type, unsafe.Pointer(fieldVal.UnsafeAddr())).Elem()
		}
		if err := r.readAutoValue(fieldVal); err != nil {
			return fmt.Errorf("binary_format: read field %s: %w", field.Name, err)
		}
	}
	return r.skipAutoPadding(int64(structAlign))
}

func (w *Writer) writeAutoValue(value reflect.Value) error {
	if !value.IsValid() {
		return fmt.Errorf("binary_format: invalid value")
	}

	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return w.WriteUint8(1)
		}
		return w.WriteUint8(0)
	case reflect.Int8:
		return w.WriteInt8(int8(value.Int()))
	case reflect.Int16:
		return w.WriteInt16(int16(value.Int()))
	case reflect.Int32:
		return w.WriteInt32(int32(value.Int()))
	case reflect.Int64:
		return w.WriteInt64(value.Int())
	case reflect.Int:
		return w.WriteInt64(value.Int())
	case reflect.Uint8:
		return w.WriteUint8(uint8(value.Uint()))
	case reflect.Uint16:
		return w.WriteUint16(uint16(value.Uint()))
	case reflect.Uint32:
		return w.WriteUint32(uint32(value.Uint()))
	case reflect.Uint64:
		return w.WriteUint64(value.Uint())
	case reflect.Uint:
		return w.WriteUint64(value.Uint())
	case reflect.Float32:
		return w.WriteFloat32(float32(value.Float()))
	case reflect.Float64:
		return w.WriteFloat64(value.Float())
	case reflect.String:
		return w.WriteStringPointer(value.String())
	case reflect.Struct:
		return w.writeAutoStruct(value)
	case reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if _, err := AlignTo(w, int64(autoTypeAlign(value.Index(i).Type()))); err != nil {
				return err
			}
			if err := w.writeAutoValue(value.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		return w.writeAutoSlice(value)
	case reflect.Map:
		return w.writeAutoMap(value)
	case reflect.Ptr:
		return w.writeAutoPointer(value)
	default:
		return fmt.Errorf("binary_format: unsupported kind %s", value.Kind())
	}
}

func (r *Reader) readAutoValue(target reflect.Value) error {
	switch target.Kind() {
	case reflect.Bool:
		v, err := r.Uint8()
		if err != nil {
			return err
		}
		target.SetBool(v != 0)
		return nil
	case reflect.Int8:
		v, err := r.Int8()
		if err != nil {
			return err
		}
		target.SetInt(int64(v))
		return nil
	case reflect.Int16:
		v, err := r.Int16()
		if err != nil {
			return err
		}
		target.SetInt(int64(v))
		return nil
	case reflect.Int32:
		v, err := r.Int32()
		if err != nil {
			return err
		}
		target.SetInt(int64(v))
		return nil
	case reflect.Int64, reflect.Int:
		v, err := r.Int64()
		if err != nil {
			return err
		}
		target.SetInt(v)
		return nil
	case reflect.Uint8:
		v, err := r.Uint8()
		if err != nil {
			return err
		}
		target.SetUint(uint64(v))
		return nil
	case reflect.Uint16:
		v, err := r.Uint16()
		if err != nil {
			return err
		}
		target.SetUint(uint64(v))
		return nil
	case reflect.Uint32:
		v, err := r.Uint32()
		if err != nil {
			return err
		}
		target.SetUint(uint64(v))
		return nil
	case reflect.Uint64, reflect.Uint:
		v, err := r.Uint64()
		if err != nil {
			return err
		}
		target.SetUint(v)
		return nil
	case reflect.Float32:
		v, err := r.Float32()
		if err != nil {
			return err
		}
		target.SetFloat(float64(v))
		return nil
	case reflect.Float64:
		v, err := r.Float64()
		if err != nil {
			return err
		}
		target.SetFloat(v)
		return nil
	case reflect.String:
		v, err := r.ReadStringPointer()
		if err != nil {
			return err
		}
		target.SetString(v)
		return nil
	case reflect.Struct:
		return r.readAutoStruct(target)
	case reflect.Array:
		for i := 0; i < target.Len(); i++ {
			if err := r.skipAutoPadding(int64(autoTypeAlign(target.Index(i).Type()))); err != nil {
				return err
			}
			if err := r.readAutoValue(target.Index(i)); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		return r.readAutoSlice(target)
	case reflect.Map:
		return r.readAutoMap(target)
	case reflect.Ptr:
		return r.readAutoPointer(target)
	default:
		return fmt.Errorf("binary_format: unsupported kind %s", target.Kind())
	}
}

func (w *Writer) writeAutoPointer(value reflect.Value) error {
	if value.IsNil() {
		return w.WriteUint64(0)
	}

	elemWriter := NewWriter(w.order)
	elemWriter.filter = w.filter
	if err := elemWriter.writeAutoValue(value.Elem()); err != nil {
		return err
	}
	if err := elemWriter.WriteDeferredData(); err != nil {
		return err
	}

	ptrLoc := w.Offset()
	blockIndex := w.deferred.AddDataBlockWithRelocationsAligned(
		elemWriter.Bytes(),
		collectRelocationPointers(elemWriter.deferred),
		autoBlockAlign(value.Elem().Type()),
	)
	w.deferred.AddPointerLocation(blockIndex, ptrLoc)
	return w.WriteUint64(0)
}

func (r *Reader) readAutoPointer(target reflect.Value) error {
	ptr, err := r.Uint64()
	if err != nil {
		return err
	}
	if ptr == 0 {
		target.SetZero()
		return nil
	}
	if ptr >= uint64(len(r.data)) {
		return fmt.Errorf("binary_format: invalid pointer offset %d", ptr)
	}

	elem := reflect.New(target.Type().Elem())
	elemReader := NewReader(r.data, r.order)
	elemReader.filter = r.filter
	elemReader.base = int64(ptr)
	elemReader.offset = int64(ptr)
	if err := elemReader.readAutoValue(elem.Elem()); err != nil {
		return err
	}
	target.Set(elem)
	return nil
}

func (w *Writer) writeAutoSlice(value reflect.Value) error {
	count := value.Len()
	elemWriter := NewWriter(w.order)
	elemWriter.filter = w.filter
	for i := 0; i < count; i++ {
		elem := value.Index(i)
		if _, err := AlignTo(elemWriter, int64(autoTypeAlign(elem.Type()))); err != nil {
			return err
		}
		if err := elemWriter.writeAutoValue(elem); err != nil {
			return err
		}
	}
	if err := elemWriter.WriteDeferredData(); err != nil {
		return err
	}

	ptrLoc := w.Offset()
	blockIndex := w.deferred.AddDataBlockWithRelocationsAligned(
		elemWriter.Bytes(),
		collectRelocationPointers(elemWriter.deferred),
		autoBlockAlign(value.Type().Elem()),
	)
	w.deferred.AddPointerLocation(blockIndex, ptrLoc)

	if err := w.WriteUint64(0); err != nil {
		return err
	}
	return w.WriteUint32(uint32(count))
}

func (r *Reader) readAutoSlice(target reflect.Value) error {
	ptr, err := r.Uint64()
	if err != nil {
		return err
	}
	count, err := r.Uint32()
	if err != nil {
		return err
	}

	if ptr == 0 || count == 0 {
		target.Set(reflect.MakeSlice(target.Type(), 0, 0))
		return nil
	}
	if ptr >= uint64(len(r.data)) {
		return fmt.Errorf("binary_format: invalid slice pointer offset %d", ptr)
	}

	elemReader := NewReader(r.data, r.order)
	elemReader.filter = r.filter
	elemReader.base = int64(ptr)
	elemReader.offset = int64(ptr)

	result := reflect.MakeSlice(target.Type(), int(count), int(count))
	for i := 0; i < int(count); i++ {
		elem := result.Index(i)
		if err := elemReader.skipAutoPadding(int64(autoTypeAlign(elem.Type()))); err != nil {
			return err
		}
		if err := elemReader.readAutoValue(elem); err != nil {
			return err
		}
	}
	target.Set(result)
	return nil
}

func (w *Writer) writeAutoMap(value reflect.Value) error {
	if value.IsNil() || value.Len() == 0 {
		if err := w.WriteUint64(0); err != nil {
			return err
		}
		if err := w.WriteUint64(0); err != nil {
			return err
		}
		return w.WriteUint32(0)
	}

	keys := value.MapKeys()
	if err := sortReflectKeys(keys); err != nil {
		return err
	}

	keyWriter := NewWriter(w.order)
	keyWriter.filter = w.filter
	valWriter := NewWriter(w.order)
	valWriter.filter = w.filter

	for _, key := range keys {
		if _, err := AlignTo(keyWriter, int64(autoTypeAlign(key.Type()))); err != nil {
			return err
		}
		if err := keyWriter.writeAutoValue(key); err != nil {
			return err
		}

		mapValue := value.MapIndex(key)
		if _, err := AlignTo(valWriter, int64(autoTypeAlign(mapValue.Type()))); err != nil {
			return err
		}
		if err := valWriter.writeAutoValue(mapValue); err != nil {
			return err
		}
	}
	if err := keyWriter.WriteDeferredData(); err != nil {
		return err
	}
	if err := valWriter.WriteDeferredData(); err != nil {
		return err
	}

	keysPtrLoc := w.Offset()
	keysBlock := w.deferred.AddDataBlockWithRelocationsAligned(
		keyWriter.Bytes(),
		collectRelocationPointers(keyWriter.deferred),
		autoTypeAlign(value.Type().Key()),
	)
	w.deferred.AddPointerLocation(keysBlock, keysPtrLoc)
	if err := w.WriteUint64(0); err != nil {
		return err
	}

	valsPtrLoc := w.Offset()
	valsBlock := w.deferred.AddDataBlockWithRelocationsAligned(
		valWriter.Bytes(),
		collectRelocationPointers(valWriter.deferred),
		autoTypeAlign(value.Type().Elem()),
	)
	w.deferred.AddPointerLocation(valsBlock, valsPtrLoc)
	if err := w.WriteUint64(0); err != nil {
		return err
	}

	return w.WriteUint32(uint32(len(keys)))
}

func (r *Reader) readAutoMap(target reflect.Value) error {
	keysPtr, err := r.Uint64()
	if err != nil {
		return err
	}
	valsPtr, err := r.Uint64()
	if err != nil {
		return err
	}
	count, err := r.Uint32()
	if err != nil {
		return err
	}

	if count == 0 {
		target.Set(reflect.MakeMap(target.Type()))
		return nil
	}
	if keysPtr >= uint64(len(r.data)) || valsPtr >= uint64(len(r.data)) {
		return fmt.Errorf("binary_format: invalid map pointer(s): keys=%d values=%d", keysPtr, valsPtr)
	}

	keysReader := NewReader(r.data, r.order)
	keysReader.filter = r.filter
	keysReader.base = int64(keysPtr)
	keysReader.offset = int64(keysPtr)

	valsReader := NewReader(r.data, r.order)
	valsReader.filter = r.filter
	valsReader.base = int64(valsPtr)
	valsReader.offset = int64(valsPtr)

	result := reflect.MakeMapWithSize(target.Type(), int(count))
	for i := 0; i < int(count); i++ {
		key := reflect.New(target.Type().Key()).Elem()
		if err := keysReader.skipAutoPadding(int64(autoTypeAlign(key.Type()))); err != nil {
			return err
		}
		if err := keysReader.readAutoValue(key); err != nil {
			return err
		}

		value := reflect.New(target.Type().Elem()).Elem()
		if err := valsReader.skipAutoPadding(int64(autoTypeAlign(value.Type()))); err != nil {
			return err
		}
		if err := valsReader.readAutoValue(value); err != nil {
			return err
		}
		result.SetMapIndex(key, value)
	}

	target.Set(result)
	return nil
}

func (r *Reader) skipAutoPadding(boundary int64) error {
	if boundary <= 0 {
		return nil
	}
	relative := (r.offset - r.base) % boundary
	if relative < 0 {
		relative += boundary
	}
	padding := (boundary - relative) % boundary
	if r.offset+padding > int64(len(r.data)) {
		return io.EOF
	}
	r.offset += padding
	return nil
}

// autoBlockAlign returns the alignment to use for the start of a deferred block
// holding a value of type t. For structs, this is the alignment of the first
// field (exported or not), matching the physical layout rule used for
// arrays-of-structs. For all other types it falls back to autoTypeAlign.
func autoBlockAlign(t reflect.Type) int {
	if t.Kind() == reflect.Struct && t.NumField() > 0 {
		return autoTypeAlign(t.Field(0).Type)
	}
	return autoTypeAlign(t)
}

func autoTypeAlign(t reflect.Type) int {
	if t == nil {
		return 1
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Int, reflect.Uint, reflect.Float64,
		reflect.Ptr, reflect.String, reflect.Slice, reflect.Map:
		return 8
	case reflect.Array:
		return autoTypeAlign(t.Elem())
	case reflect.Struct:
		maxAlign := 1
		for i := 0; i < t.NumField(); i++ {
			fieldAlign := autoTypeAlign(t.Field(i).Type)
			if fieldAlign > maxAlign {
				maxAlign = fieldAlign
			}
		}
		if maxAlign > 8 {
			return 8
		}
		return maxAlign
	default:
		return 1
	}
}

func sortReflectKeys(keys []reflect.Value) error {
	if len(keys) == 0 {
		return nil
	}

	kind := keys[0].Kind()
	for i := 1; i < len(keys); i++ {
		if keys[i].Kind() != kind {
			return fmt.Errorf("binary_format: mixed map key kinds are not supported")
		}
	}

	switch kind {
	case reflect.String:
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sort.Slice(keys, func(i, j int) bool { return keys[i].Int() < keys[j].Int() })
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		sort.Slice(keys, func(i, j int) bool { return keys[i].Uint() < keys[j].Uint() })
	case reflect.Bool:
		sort.Slice(keys, func(i, j int) bool { return !keys[i].Bool() && keys[j].Bool() })
	default:
		return fmt.Errorf("binary_format: unsupported map key type %s for auto serialization", kind)
	}
	return nil
}

func collectRelocationPointers(deferred *DeferredDataBlocks) []int64 {
	if deferred == nil {
		return nil
	}
	seen := make(map[int64]struct{})
	locations := make([]int64, 0)
	for _, block := range deferred.blocks {
		for _, loc := range block.PointerLocations {
			if _, ok := seen[loc]; ok {
				continue
			}
			seen[loc] = struct{}{}
			locations = append(locations, loc)
		}
	}
	sort.Slice(locations, func(i, j int) bool { return locations[i] < locations[j] })
	return locations
}
