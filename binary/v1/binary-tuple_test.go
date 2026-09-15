package ignite3

import (
	"bytes"
	"math/big"
	"testing"
	"time"
)

// Layout check against a hand-computed tuple: INT32 1, STRING "ab".
// Value area = 1 byte + 2 bytes = 3 bytes -> entry size 1, header 0x00.
func TestBuildBinaryTupleLayout(t *testing.T) {
	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtInt32}, {Kind: BtString}},
		[]interface{}{int32(1), "ab"},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	want := []byte{0x00, 0x01, 0x03, 0x01, 'a', 'b'}
	if !bytes.Equal(got, want) {
		t.Fatalf("layout mismatch\n got=% x\nwant=% x", got, want)
	}
}

// NULL must occupy zero bytes, so the offset stays equal to the previous one.
func TestBuildBinaryTupleNull(t *testing.T) {
	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtInt32}, {Kind: BtInt32}, {Kind: BtInt32}},
		[]interface{}{int32(7), nil, int32(8)},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	want := []byte{0x00, 0x01, 0x01, 0x02, 0x07, 0x08}
	if !bytes.Equal(got, want) {
		t.Fatalf("null layout mismatch\n got=% x\nwant=% x", got, want)
	}
}

// Empty strings are encoded as a single 0x80 byte, so they are distinguishable from NULL.
func TestBuildBinaryTupleEmptyString(t *testing.T) {
	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtString}, {Kind: BtString}},
		[]interface{}{"", nil},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	want := []byte{0x00, 0x01, 0x01, 0x80}
	if !bytes.Equal(got, want) {
		t.Fatalf("empty string layout mismatch\n got=% x\nwant=% x", got, want)
	}
}

// A value area larger than 255 bytes must switch the offset table to 2-byte entries.
func TestBuildBinaryTupleBigOffsetTable(t *testing.T) {
	long := ""
	for i := 0; i < 300; i++ {
		long += "x"
	}

	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtString}, {Kind: BtInt32}},
		[]interface{}{long, int32(1)},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	if got[0] != 0b01 {
		t.Fatalf("expected 2-byte offset table header, got %#x", got[0])
	}
	// header(1) + 2 entries * 2 bytes + value area(300 + 1).
	if want := 1 + 4 + 301; len(got) != want {
		t.Fatalf("length mismatch: got %d, want %d", len(got), want)
	}
}

// Doubles that are exactly representable as float must be written in 4 bytes (Java behaviour).
func TestBuildBinaryTupleDoubleShrink(t *testing.T) {
	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtDouble}, {Kind: BtDouble}},
		[]interface{}{1.5, 1.1},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	// header + 2 offsets, then 4 bytes (1.5 as float) + 8 bytes (1.1 as double).
	if want := 1 + 2 + 4 + 8; len(got) != want {
		t.Fatalf("length mismatch: got %d, want %d", len(got), want)
	}
}

// Timestamps with zero nanoseconds take 8 bytes, with nanoseconds 12 bytes.
func TestBuildBinaryTupleTimestamp(t *testing.T) {
	base := time.Date(2026, 4, 20, 16, 29, 40, 0, time.UTC)
	withNanos := time.Date(2026, 4, 20, 16, 29, 40, 644838000, time.UTC)

	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtTimestamp}, {Kind: BtTimestamp}},
		[]interface{}{base, withNanos},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	if want := 1 + 2 + 8 + 12; len(got) != want {
		t.Fatalf("length mismatch: got %d, want %d", len(got), want)
	}
}

// DECIMAL: int16 scale followed by the two's-complement unscaled value.
func TestBuildBinaryTupleDecimal(t *testing.T) {
	got, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtDecimal, Scale: 2}},
		[]interface{}{Decimal{Unscaled: big.NewInt(12345), Scale: 2}},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}

	want := []byte{0x00, 0x04, 0x02, 0x00, 0x30, 0x39}
	if !bytes.Equal(got, want) {
		t.Fatalf("decimal layout mismatch\n got=% x\nwant=% x", got, want)
	}

	neg, err := BuildBinaryTuple(
		[]BinaryTupleColumn{{Kind: BtDecimal, Scale: 0}},
		[]interface{}{Decimal{Unscaled: big.NewInt(-1), Scale: 0}},
	)
	if err != nil {
		t.Fatalf("BuildBinaryTuple: %v", err)
	}
	if neg[len(neg)-1] != 0xff {
		t.Fatalf("expected -1 to be encoded as 0xff, got %#x", neg[len(neg)-1])
	}
}

func TestBuildBinaryTupleLengthMismatch(t *testing.T) {
	_, err := BuildBinaryTuple([]BinaryTupleColumn{{Kind: BtInt32}}, nil)
	if err == nil {
		t.Fatal("expected an error when the number of columns and values differ")
	}
}