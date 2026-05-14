package binary_format

import (
	"bytes"
	stdbinary "encoding/binary"
	"math"
	"reflect"
	"testing"
)

type testChild struct {
	X int16
	Y int16
}

type autoPacket struct {
	Version uint16
	Name    string
	Values  []uint32
	Tags    map[string]uint16
	Child   testChild
	Note    *uint32
}

func TestWriterReaderEndianness(t *testing.T) {
	writer := NewWriter(stdbinary.BigEndian)

	if err := writer.WriteUint16(0x1234); err != nil {
		t.Fatalf("WriteUint16 failed: %v", err)
	}
	if err := writer.WriteUint32(0x01020304); err != nil {
		t.Fatalf("WriteUint32 failed: %v", err)
	}
	if err := writer.WriteFloat32(1.5); err != nil {
		t.Fatalf("WriteFloat32 failed: %v", err)
	}

	result := writer.Bytes()
	expected := []byte{0x12, 0x34, 0x01, 0x02, 0x03, 0x04, 0x3f, 0xc0, 0x00, 0x00}
	if !bytes.Equal(result, expected) {
		t.Fatalf("unexpected bytes: got %v want %v", result, expected)
	}
	if writer.Offset() != int64(len(expected)) {
		t.Fatalf("unexpected writer offset: got %d want %d", writer.Offset(), len(expected))
	}

	reader := NewReader(result, stdbinary.BigEndian)
	value16, err := reader.Uint16()
	if err != nil {
		t.Fatalf("Uint16 failed: %v", err)
	}
	value32, err := reader.Uint32()
	if err != nil {
		t.Fatalf("Uint32 failed: %v", err)
	}
	valueFloat, err := reader.Float32()
	if err != nil {
		t.Fatalf("Float32 failed: %v", err)
	}

	if value16 != 0x1234 {
		t.Fatalf("unexpected uint16: got %#x want %#x", value16, 0x1234)
	}
	if value32 != 0x01020304 {
		t.Fatalf("unexpected uint32: got %#x want %#x", value32, 0x01020304)
	}
	if math.Float32bits(valueFloat) != math.Float32bits(1.5) {
		t.Fatalf("unexpected float32: got %v want %v", valueFloat, 1.5)
	}
	if reader.Offset() != int64(len(expected)) {
		t.Fatalf("unexpected reader offset: got %d want %d", reader.Offset(), len(expected))
	}
}

func TestAutoReflectionRoundTrip(t *testing.T) {
	note := uint32(1234)
	input := autoPacket{
		Version: 12,
		Name:    "auto",
		Values:  []uint32{5, 6, 7},
		Tags: map[string]uint16{
			"left":  10,
			"right": 20,
		},
		Child: testChild{X: -9, Y: 14},
		Note:  &note,
	}

	writer := NewWriter(stdbinary.LittleEndian)
	if err := writer.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}
	if err := writer.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	reader := NewReader(writer.Bytes(), stdbinary.LittleEndian)
	var output autoPacket
	if err := reader.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}

	if output.Note == nil || *output.Note != *input.Note {
		t.Fatalf("note mismatch: got %#v want %#v", output.Note, input.Note)
	}
	if output.Version != input.Version {
		t.Fatalf("version mismatch: got %d want %d", output.Version, input.Version)
	}
	if output.Name != input.Name {
		t.Fatalf("name mismatch: got %q want %q", output.Name, input.Name)
	}
	if !reflect.DeepEqual(output.Values, input.Values) {
		t.Fatalf("values mismatch: got %v want %v", output.Values, input.Values)
	}
	if !reflect.DeepEqual(output.Tags, input.Tags) {
		t.Fatalf("tags mismatch: got %v want %v", output.Tags, input.Tags)
	}
	if !reflect.DeepEqual(output.Child, input.Child) {
		t.Fatalf("child mismatch: got %#v want %#v", output.Child, input.Child)
	}
}

func TestAutoReflectionFieldFilterByName(t *testing.T) {
	note := uint32(77)
	input := autoPacket{
		Version: 3,
		Name:    "filtered",
		Values:  []uint32{1, 2},
		Tags: map[string]uint16{
			"a": 1,
		},
		Child: testChild{X: 1, Y: 2},
		Note:  &note,
	}

	filter := func(_ reflect.Type, field reflect.StructField) bool {
		return field.Name != "Name" && field.Name != "Tags"
	}

	writer := NewWriter(stdbinary.LittleEndian)
	writer.SetFieldFilter(filter)
	if err := writer.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}
	if err := writer.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	reader := NewReader(writer.Bytes(), stdbinary.LittleEndian)
	reader.SetFieldFilter(filter)
	var output autoPacket
	if err := reader.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}

	if output.Name != "" {
		t.Fatalf("expected Name to be filtered out, got %q", output.Name)
	}
	if len(output.Tags) != 0 {
		t.Fatalf("expected Tags to be filtered out, got %v", output.Tags)
	}
	if output.Version != input.Version {
		t.Fatalf("version mismatch: got %d want %d", output.Version, input.Version)
	}
	if !reflect.DeepEqual(output.Values, input.Values) {
		t.Fatalf("values mismatch: got %v want %v", output.Values, input.Values)
	}
	if !reflect.DeepEqual(output.Child, input.Child) {
		t.Fatalf("child mismatch: got %#v want %#v", output.Child, input.Child)
	}
	if output.Note == nil || *output.Note != *input.Note {
		t.Fatalf("note mismatch: got %#v want %#v", output.Note, input.Note)
	}
}

func TestBufferedWriterDeferredStringPointers(t *testing.T) {
	w := NewWriter(stdbinary.LittleEndian)

	// Write a simple structure with deferred strings.
	if err := w.WriteUint32(42); err != nil {
		t.Fatalf("WriteUint32 failed: %v", err)
	}

	// Write first deferred string: "hello"
	if err := w.WriteStringPointer("hello"); err != nil {
		t.Fatalf("WriteStringPointer 1 failed: %v", err)
	}

	// Write second deferred string: "world"
	if err := w.WriteStringPointer("world"); err != nil {
		t.Fatalf("WriteStringPointer 2 failed: %v", err)
	}

	// Finalize: write deferred data and update pointers.
	if err := w.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	result := w.Bytes()

	// Verify structure:
	// Bytes 0-3: uint32(42)
	// Bytes 4-11: pointer to "hello" (8 bytes)
	// Bytes 12-19: uint32(5) + uint32(5) (byte count + char count)
	// Bytes 20-27: pointer to "world" (8 bytes)
	// Bytes 28-35: uint32(5) + uint32(5) (byte count + char count)
	// Then deferred data: "helloworld"

	// Check first 4 bytes are 42.
	if stdbinary.LittleEndian.Uint32(result[0:4]) != 42 {
		t.Fatalf("unexpected initial uint32: got %d want 42", stdbinary.LittleEndian.Uint32(result[0:4]))
	}

	// Extract first pointer.
	ptr1 := stdbinary.LittleEndian.Uint64(result[4:12])
	if ptr1 == 0 {
		t.Fatalf("first pointer not updated: got 0")
	}

	// Verify byte and char counts for first string.
	byteCnt1 := stdbinary.LittleEndian.Uint32(result[12:16])
	charCnt1 := stdbinary.LittleEndian.Uint32(result[16:20])
	if byteCnt1 != 5 || charCnt1 != 5 {
		t.Fatalf("first string counts: got (%d, %d) want (5, 5)", byteCnt1, charCnt1)
	}

	// Extract second pointer.
	ptr2 := stdbinary.LittleEndian.Uint64(result[20:28])
	if ptr2 == 0 {
		t.Fatalf("second pointer not updated: got 0")
	}

	// Verify byte and char counts for second string.
	byteCnt2 := stdbinary.LittleEndian.Uint32(result[28:32])
	charCnt2 := stdbinary.LittleEndian.Uint32(result[32:36])
	if byteCnt2 != 5 || charCnt2 != 5 {
		t.Fatalf("second string counts: got (%d, %d) want (5, 5)", byteCnt2, charCnt2)
	}

	// Check that deferred data is present and correct.
	str1Data := result[ptr1 : ptr1+5]
	str2Data := result[ptr2 : ptr2+5]
	if string(str1Data) != "hello" {
		t.Fatalf("string 1 data: got %q want %q", string(str1Data), "hello")
	}
	if string(str2Data) != "world" {
		t.Fatalf("string 2 data: got %q want %q", string(str2Data), "world")
	}
}

func TestDeferredBlockAlignmentForUint64Slice(t *testing.T) {
	type packet struct {
		Name   string
		Values []uint64
	}

	input := packet{
		Name:   "abcde",
		Values: []uint64{11, 22, 33},
	}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}
	if err := w.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	result := w.Bytes()

	namePtr := stdbinary.LittleEndian.Uint64(result[0:8])
	valuesPtr := stdbinary.LittleEndian.Uint64(result[16:24])
	if valuesPtr%8 != 0 {
		t.Fatalf("values block pointer is not 8-byte aligned: got %d", valuesPtr)
	}
	if valuesPtr < namePtr+5 {
		t.Fatalf("values block pointer overlaps previous deferred data: namePtr=%d valuesPtr=%d", namePtr, valuesPtr)
	}

	reader := NewReader(result, stdbinary.LittleEndian)
	var output packet
	if err := reader.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}

	if output.Name != input.Name {
		t.Fatalf("name mismatch: got %q want %q", output.Name, input.Name)
	}
	if !reflect.DeepEqual(output.Values, input.Values) {
		t.Fatalf("values mismatch: got %v want %v", output.Values, input.Values)
	}
}

func TestBufferedWriterWithStructAndDeferredFields(t *testing.T) {
	type metadata struct {
		Title string
		Count uint32
	}

	w := NewWriter(stdbinary.LittleEndian)

	// Manually write a struct with deferred string field.
	meta := metadata{Title: "test", Count: 99}

	// Write deferred string field.
	if err := w.WriteStringPointer(meta.Title); err != nil {
		t.Fatalf("WriteStringPointer failed: %v", err)
	}

	// Write count field.
	if err := w.WriteUint32(meta.Count); err != nil {
		t.Fatalf("WriteUint32 failed: %v", err)
	}

	// Finalize.
	if err := w.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	result := w.Bytes()

	// Verify: pointer (8) + bytecount (4) + charcount (4) + count (4) + deferred data
	ptr := stdbinary.LittleEndian.Uint64(result[0:8])
	if ptr == 0 {
		t.Fatalf("deferred pointer not updated")
	}

	byteCnt := stdbinary.LittleEndian.Uint32(result[8:12])
	charCnt := stdbinary.LittleEndian.Uint32(result[12:16])
	count := stdbinary.LittleEndian.Uint32(result[16:20])

	if byteCnt != 4 || charCnt != 4 {
		t.Fatalf("string counts: got (%d, %d) want (4, 4)", byteCnt, charCnt)
	}
	if count != 99 {
		t.Fatalf("count: got %d want 99", count)
	}

	strData := result[ptr : ptr+4]
	if string(strData) != "test" {
		t.Fatalf("deferred string data: got %q want %q", string(strData), "test")
	}
}

func TestAlignTo(t *testing.T) {
	w := NewWriter(stdbinary.LittleEndian)

	// Write 5 bytes, then align to 8.
	w.WriteBytes([]byte{1, 2, 3, 4, 5})

	padding, err := AlignTo(w, 8)
	if err != nil {
		t.Fatalf("AlignTo failed: %v", err)
	}

	if padding != 3 {
		t.Fatalf("unexpected padding: got %d want 3", padding)
	}

	if w.Offset() != 8 {
		t.Fatalf("unexpected offset after alignment: got %d want 8", w.Offset())
	}

	result := w.Bytes()
	if len(result) != 8 {
		t.Fatalf("unexpected buffer size: got %d want 8", len(result))
	}

	// Verify padding is zeros.
	if result[5] != 0 || result[6] != 0 || result[7] != 0 {
		t.Fatalf("unexpected padding bytes: got %v want [0, 0, 0]", result[5:8])
	}
}

// structWithUnexported has a mix of exported and unexported fields.
// The unexported fields must be written and read back correctly via unsafe.
type structWithUnexported struct {
	A uint32
	b uint16 // unexported
	C uint8
	d int8 // unexported
}

func TestAutoReflectionUnexportedFields(t *testing.T) {
	input := structWithUnexported{A: 0xDEAD, b: 0xBEEF, C: 42, d: -7}

	writer := NewWriter(stdbinary.LittleEndian)
	if err := writer.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}
	if err := writer.WriteDeferredData(); err != nil {
		t.Fatalf("WriteDeferredData failed: %v", err)
	}

	reader := NewReader(writer.Bytes(), stdbinary.LittleEndian)
	var output structWithUnexported
	if err := reader.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}

	if output.A != input.A {
		t.Fatalf("A mismatch: got %d want %d", output.A, input.A)
	}
	if output.b != input.b {
		t.Fatalf("b mismatch: got %d want %d", output.b, input.b)
	}
	if output.C != input.C {
		t.Fatalf("C mismatch: got %d want %d", output.C, input.C)
	}
	if output.d != input.d {
		t.Fatalf("d mismatch: got %d want %d", output.d, input.d)
	}
}

// ---------------------------------------------------------------------------
// Field alignment tests
//
// writeAutoStruct inserts padding before each field so its offset is a
// multiple of the field's natural alignment, and appends trailing padding so
// the total struct size is a multiple of the struct's alignment (the maximum
// alignment of any field).  The tests below pin the exact byte layout to
// guard against regressions.
// ---------------------------------------------------------------------------

// alignedStruct1: uint8 → 1-byte pad → uint16
// Expected layout (4 bytes): [A][0][B_lo B_hi]
// Struct align = 2, total = 4 bytes.
type alignedStruct1 struct {
	A uint8
	B uint16
}

func TestFieldAlignmentUint8ThenUint16(t *testing.T) {
	input := alignedStruct1{A: 0xAB, B: 0x1234}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 4 // 1 (A) + 1 (pad) + 2 (B)
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if result[0] != 0xAB {
		t.Fatalf("A: got %#x want %#x", result[0], 0xAB)
	}
	if result[1] != 0x00 {
		t.Fatalf("padding byte at offset 1: got %#x want 0x00", result[1])
	}
	if got := stdbinary.LittleEndian.Uint16(result[2:4]); got != 0x1234 {
		t.Fatalf("B: got %#x want %#x", got, 0x1234)
	}

	// Round-trip.
	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct1
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct2: uint8 → 3-byte pad → uint32
// Expected layout (8 bytes): [A][0 0 0][B×4]
// Struct align = 4, total = 8 bytes.
type alignedStruct2 struct {
	A uint8
	B uint32
}

func TestFieldAlignmentUint8ThenUint32(t *testing.T) {
	input := alignedStruct2{A: 0x01, B: 0xDEADBEEF}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 8 // 1 (A) + 3 (pad) + 4 (B)
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if result[0] != 0x01 {
		t.Fatalf("A: got %#x want 0x01", result[0])
	}
	for i := 1; i <= 3; i++ {
		if result[i] != 0x00 {
			t.Fatalf("padding byte at offset %d: got %#x want 0x00", i, result[i])
		}
	}
	if got := stdbinary.LittleEndian.Uint32(result[4:8]); got != 0xDEADBEEF {
		t.Fatalf("B: got %#x want %#x", got, uint32(0xDEADBEEF))
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct2
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct3: uint8 → 7-byte pad → uint64
// Expected layout (16 bytes): [A][0×7][B×8]
// Struct align = 8, total = 16 bytes.
type alignedStruct3 struct {
	A uint8
	B uint64
}

func TestFieldAlignmentUint8ThenUint64(t *testing.T) {
	input := alignedStruct3{A: 0xFF, B: 0x0102030405060708}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 16 // 1 (A) + 7 (pad) + 8 (B)
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if result[0] != 0xFF {
		t.Fatalf("A: got %#x want 0xff", result[0])
	}
	for i := 1; i <= 7; i++ {
		if result[i] != 0x00 {
			t.Fatalf("padding byte at offset %d: got %#x want 0x00", i, result[i])
		}
	}
	if got := stdbinary.LittleEndian.Uint64(result[8:16]); got != 0x0102030405060708 {
		t.Fatalf("B: got %#x want %#x", got, uint64(0x0102030405060708))
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct3
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct4: uint16 → 2-byte pad → uint32
// Expected layout (8 bytes): [A×2][0 0][B×4]
// Struct align = 4, total = 8 bytes.
type alignedStruct4 struct {
	A uint16
	B uint32
}

func TestFieldAlignmentUint16ThenUint32(t *testing.T) {
	input := alignedStruct4{A: 0xCAFE, B: 0xBEEFDEAD}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 8 // 2 (A) + 2 (pad) + 4 (B)
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if got := stdbinary.LittleEndian.Uint16(result[0:2]); got != 0xCAFE {
		t.Fatalf("A: got %#x want %#x", got, uint16(0xCAFE))
	}
	if result[2] != 0x00 || result[3] != 0x00 {
		t.Fatalf("padding bytes at offsets 2-3: got %v want [0 0]", result[2:4])
	}
	if got := stdbinary.LittleEndian.Uint32(result[4:8]); got != 0xBEEFDEAD {
		t.Fatalf("B: got %#x want %#x", got, uint32(0xBEEFDEAD))
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct4
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct5: uint32 then uint8 — trailing pad required.
// Expected layout (8 bytes): [A×4][B][0 0 0]
// Struct align = 4, so 3 trailing padding bytes are added after B.
type alignedStruct5 struct {
	A uint32
	B uint8
}

func TestFieldAlignmentTrailingPadding(t *testing.T) {
	input := alignedStruct5{A: 0x11223344, B: 0x77}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 8 // 4 (A) + 1 (B) + 3 (trailing pad)
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if got := stdbinary.LittleEndian.Uint32(result[0:4]); got != 0x11223344 {
		t.Fatalf("A: got %#x want %#x", got, uint32(0x11223344))
	}
	if result[4] != 0x77 {
		t.Fatalf("B: got %#x want 0x77", result[4])
	}
	for i := 5; i <= 7; i++ {
		if result[i] != 0x00 {
			t.Fatalf("trailing padding byte at offset %d: got %#x want 0x00", i, result[i])
		}
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct5
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct6: ascending widths uint8, uint16, uint32, uint64.
// Expected layout (16 bytes):
//   offset 0:  A (1 byte)
//   offset 1:  pad (1 byte, align to 2 for B)
//   offset 2:  B (2 bytes)
//   offset 4:  C (4 bytes, already aligned)
//   offset 8:  D (8 bytes, already aligned)
//   total: 16 bytes, trailing pad: 0
type alignedStruct6 struct {
	A uint8
	B uint16
	C uint32
	D uint64
}

func TestFieldAlignmentAscendingWidths(t *testing.T) {
	input := alignedStruct6{A: 0x01, B: 0x0203, C: 0x04050607, D: 0x08090A0B0C0D0E0F}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 16
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if result[0] != 0x01 {
		t.Fatalf("A: got %#x want 0x01", result[0])
	}
	if result[1] != 0x00 {
		t.Fatalf("pad at offset 1: got %#x want 0x00", result[1])
	}
	if got := stdbinary.LittleEndian.Uint16(result[2:4]); got != 0x0203 {
		t.Fatalf("B: got %#x want %#x", got, uint16(0x0203))
	}
	if got := stdbinary.LittleEndian.Uint32(result[4:8]); got != 0x04050607 {
		t.Fatalf("C: got %#x want %#x", got, uint32(0x04050607))
	}
	if got := stdbinary.LittleEndian.Uint64(result[8:16]); got != 0x08090A0B0C0D0E0F {
		t.Fatalf("D: got %#x want %#x", got, uint64(0x08090A0B0C0D0E0F))
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct6
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedInner: inner struct { X uint8; Y uint32 }
// alignedOuter: outer struct { A uint8; B alignedInner }
//
// Outer align = max(1, inner_align) = 4.
// Expected layout (12 bytes):
//   offset 0:  A (1 byte)
//   offset 1:  pad (3 bytes, align to inner align = 4)
//   offset 4:  inner.X (1 byte)
//   offset 5:  pad (3 bytes, align to 4 for inner.Y)
//   offset 8:  inner.Y (4 bytes)
//   trailing pad to outer align (4): 0 bytes
//   total: 12 bytes
type alignedInner struct {
	X uint8
	Y uint32
}

type alignedOuter struct {
	A uint8
	B alignedInner
}

func TestFieldAlignmentNestedStruct(t *testing.T) {
	input := alignedOuter{A: 0xAA, B: alignedInner{X: 0xBB, Y: 0xCCDDEEFF}}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 12 // 1+3+1+3+4
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes, got %d: %v", wantLen, len(result), result)
	}
	if result[0] != 0xAA {
		t.Fatalf("A: got %#x want 0xaa", result[0])
	}
	for i := 1; i <= 3; i++ {
		if result[i] != 0x00 {
			t.Fatalf("inter-field padding byte at offset %d: got %#x want 0x00", i, result[i])
		}
	}
	if result[4] != 0xBB {
		t.Fatalf("B.X: got %#x want 0xbb", result[4])
	}
	for i := 5; i <= 7; i++ {
		if result[i] != 0x00 {
			t.Fatalf("inner padding byte at offset %d: got %#x want 0x00", i, result[i])
		}
	}
	if got := stdbinary.LittleEndian.Uint32(result[8:12]); got != 0xCCDDEEFF {
		t.Fatalf("B.Y: got %#x want %#x", got, uint32(0xCCDDEEFF))
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedOuter
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}

// alignedStruct7: multiple same-size fields — no inter-field padding expected.
// struct { A uint32; B uint32; C uint32 }
// Expected layout (12 bytes): [A×4][B×4][C×4], trailing pad to 4: 0 bytes.
type alignedStruct7 struct {
	A uint32
	B uint32
	C uint32
}

func TestFieldAlignmentNoExtraPaddingForSameSize(t *testing.T) {
	input := alignedStruct7{A: 1, B: 2, C: 3}

	w := NewWriter(stdbinary.LittleEndian)
	if err := w.WriteAuto(input); err != nil {
		t.Fatalf("WriteAuto failed: %v", err)
	}

	result := w.Bytes()
	const wantLen = 12
	if len(result) != wantLen {
		t.Fatalf("expected %d bytes (no padding), got %d: %v", wantLen, len(result), result)
	}
	if got := stdbinary.LittleEndian.Uint32(result[0:4]); got != 1 {
		t.Fatalf("A: got %d want 1", got)
	}
	if got := stdbinary.LittleEndian.Uint32(result[4:8]); got != 2 {
		t.Fatalf("B: got %d want 2", got)
	}
	if got := stdbinary.LittleEndian.Uint32(result[8:12]); got != 3 {
		t.Fatalf("C: got %d want 3", got)
	}

	r := NewReader(result, stdbinary.LittleEndian)
	var output alignedStruct7
	if err := r.ReadAuto(&output); err != nil {
		t.Fatalf("ReadAuto failed: %v", err)
	}
	if output != input {
		t.Fatalf("round-trip mismatch: got %+v want %+v", output, input)
	}
}
