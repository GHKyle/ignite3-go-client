package ignite3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"
)

// Binary tuple (IEP-92) encoding, matching org.apache.ignite.internal.binarytuple.BinaryTupleBuilder
// from the Apache Ignite 3 Java client. Used by the tuple (record/view) operations such as
// OP_TUPLE_UPSERT_ALL, where rows are sent as binary tuples instead of SQL parameters.
//
// Layout:
//
//		+--------+----------------+--------------+
//		| header | offset table   | value area   |
//		+--------+----------------+--------------+
//
//	  - header: 1 byte. Bits 0..1 hold log2(entry size) of the offset table, so the entry size is
//	    1<<(header&0b011) bytes (1, 2 or 4).
//	  - offset table: numElements entries, little-endian. Entry i holds the END offset of element i
//	    inside the value area (a running total), -1 meaning "not set".
//	  - value area: element payloads, concatenated in element order.
//
// A NULL element occupies zero bytes, so its offset equals the previous element offset.
// An empty STRING / BYTE_ARRAY is encoded with a leading 0x80 byte, therefore those types never
// have a zero-length payload - which is what makes "zero length" an unambiguous NULL marker.
const (
	// BtNull is a NULL element.
	BtNull = iota
	// BtBoolean is a BOOLEAN element (1 byte).
	BtBoolean
	// BtInt8 is a TINYINT element.
	BtInt8
	// BtInt16 is a SMALLINT element.
	BtInt16
	// BtInt32 is an INT element.
	BtInt32
	// BtInt64 is a BIGINT element.
	BtInt64
	// BtFloat is a REAL element.
	BtFloat
	// BtDouble is a DOUBLE element.
	BtDouble
	// BtDecimal is a DECIMAL element: int16 scale (LE) followed by the big-endian two's-complement
	// unscaled value.
	BtDecimal
	// BtDate is a DATE element (3 bytes).
	BtDate
	// BtTime is a TIME element (4 or 5 bytes).
	BtTime
	// BtDateTime is a local TIMESTAMP (DATETIME) element: date bytes followed by time bytes.
	BtDateTime
	// BtTimestamp is an instant TIMESTAMP element: int64 epoch seconds (LE), optionally followed by
	// int32 nanoseconds (LE) when they are not zero.
	BtTimestamp
	// BtUuid is a UUID element: two int64 values (LE).
	BtUuid
	// BtString is a VARCHAR element (UTF-8, empty string is encoded as 0x80).
	BtString
	// BtBytes is a VARBINARY element (prefixed with 0x80 when empty or when it starts with 0x80).
	BtBytes
)

// varlenEmptyByte marks an empty variable-length element.
const varlenEmptyByte = byte(0x80)

// BinaryTupleColumn describes a single column of a binary tuple row.
type BinaryTupleColumn struct {
	// Kind is one of the Bt* constants.
	Kind int
	// Scale is used for BtDecimal only.
	Scale int
}

// BuildBinaryTuple encodes one row as an Ignite 3 binary tuple.
//
// cols and vals must have the same length. Supported Go value types are: nil, bool,
// int / int8..int64 / uint8..uint64, float32, float64, string, []byte, ignite3.Decimal,
// ignite3.Date, ignite3.Time, ignite3.DateTime, ignite3.Timestamp, time.Time and ignite3.Uuid.
// A nil value (or a value that cannot be converted) is written as a NULL element.
func BuildBinaryTuple(cols []BinaryTupleColumn, vals []interface{}) ([]byte, error) {
	if len(cols) != len(vals) {
		return nil, fmt.Errorf("binary tuple: %d columns but %d values", len(cols), len(vals))
	}

	// Build the offset table and the value area in one pass.
	var valueArea bytes.Buffer
	offsets := make([]int32, len(cols))

	for i, col := range cols {
		if err := appendBinaryTupleValue(&valueArea, col, vals[i]); err != nil {
			return nil, fmt.Errorf("binary tuple: column %d: %w", i, err)
		}
		offsets[i] = int32(valueArea.Len())
	}

	entrySize := 1
	switch {
	case valueArea.Len() <= 0xff:
		entrySize = 1
	case valueArea.Len() <= 0xffff:
		entrySize = 2
	case int64(valueArea.Len()) <= math.MaxInt32:
		entrySize = 4
	default:
		return nil, fmt.Errorf("binary tuple: value area is too big: %d bytes", valueArea.Len())
	}

	out := make([]byte, 0, 1+entrySize*len(cols)+valueArea.Len())

	// Header: log2(entrySize) in the lowest two bits.
	switch entrySize {
	case 1:
		out = append(out, 0b00)
	case 2:
		out = append(out, 0b01)
	default:
		out = append(out, 0b10)
	}

	// Offset table, little-endian, entry i = end offset of element i.
	raw := make([]byte, 4)
	for _, off := range offsets {
		binary.LittleEndian.PutUint32(raw, uint32(off))
		out = append(out, raw[:entrySize]...)
	}

	out = append(out, valueArea.Bytes()...)
	return out, nil
}

// appendBinaryTupleValue encodes a single element, mirroring the Java BinaryTupleBuilder
// append* methods (integers are written in the narrowest width that fits, doubles fall back
// to float when that is lossless, and so on).
func appendBinaryTupleValue(buf *bytes.Buffer, col BinaryTupleColumn, val interface{}) error {
	if val == nil {
		// NULL: zero bytes.
		return nil
	}

	switch col.Kind {
	case BtNull:
		return nil

	case BtBoolean:
		b, err := toBool(val)
		if err != nil {
			return err
		}
		if b {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
		return nil

	case BtInt8, BtInt16, BtInt32, BtInt64:
		n, err := toInt64(val)
		if err != nil {
			return err
		}
		return putBinaryTupleInt(buf, col.Kind, n)

	case BtFloat:
		f, err := toFloat64(val)
		if err != nil {
			return err
		}
		putUint32(buf, math.Float32bits(float32(f)))
		return nil

	case BtDouble:
		f, err := toFloat64(val)
		if err != nil {
			return err
		}
		// Java: appendDouble falls back to appendFloat when the value is exactly representable.
		if float64(float32(f)) == f {
			putUint32(buf, math.Float32bits(float32(f)))
			return nil
		}
		putUint64(buf, math.Float64bits(f))
		return nil

	case BtDecimal:
		d, err := toDecimal(val, col.Scale)
		if err != nil {
			return err
		}
		var scaleBytes [2]byte
		binary.LittleEndian.PutUint16(scaleBytes[:], uint16(int16(d.Scale)))
		buf.Write(scaleBytes[:])
		buf.Write(twosComplementBytes(d.Unscaled))
		return nil

	case BtDate:
		t, err := toTime(val)
		if err != nil {
			return err
		}
		dat, err := encodeDate(t.Year(), int(t.Month()), t.Day())
		if err != nil {
			return err
		}
		buf.Write(dat)
		return nil

	case BtTime:
		t, err := toTime(val)
		if err != nil {
			return err
		}
		b, err := encodeTime(t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil

	case BtDateTime:
		t, err := toTime(val)
		if err != nil {
			return err
		}
		dat, err := encodeDate(t.Year(), int(t.Month()), t.Day())
		if err != nil {
			return err
		}
		buf.Write(dat)
		b, err := encodeTime(t.Hour(), t.Minute(), t.Second(), t.Nanosecond())
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil

	case BtTimestamp:
		t, err := toTime(val)
		if err != nil {
			return err
		}
		putUint64(buf, uint64(t.Unix()))
		// Java appends nanoseconds only when they are non-zero.
		if ns := t.Nanosecond(); ns != 0 {
			putUint32(buf, uint32(ns))
		}
		return nil

	case BtUuid:
		u, ok := val.(Uuid)
		if !ok {
			if pu, ok2 := val.(*Uuid); ok2 && pu != nil {
				u = *pu
			} else {
				return fmt.Errorf("cannot convert %T to UUID", val)
			}
		}
		putUint64(buf, binary.LittleEndian.Uint64(u.UUID[0:8]))
		putUint64(buf, binary.LittleEndian.Uint64(u.UUID[8:16]))
		return nil

	case BtString:
		s, ok := val.(string)
		if !ok {
			s = fmt.Sprintf("%v", val)
		}
		if s == "" {
			// Empty string: single 0x80 byte.
			buf.WriteByte(varlenEmptyByte)
			return nil
		}
		buf.WriteString(s)
		return nil

	case BtBytes:
		var b []byte
		switch v := val.(type) {
		case []byte:
			b = v
		case string:
			b = []byte(v)
		default:
			return fmt.Errorf("cannot convert %T to byte array", val)
		}
		// Java: prefix with 0x80 when empty or when the payload itself starts with 0x80.
		if len(b) == 0 || b[0] == varlenEmptyByte {
			buf.WriteByte(varlenEmptyByte)
		}
		buf.Write(b)
		return nil
	}

	return fmt.Errorf("unsupported binary tuple kind: %d", col.Kind)
}

// putBinaryTupleInt writes an integer using the narrowest width accepted by the Java builder:
// for INT the order is byte, short, int; for BIGINT it is short, int, long.
func putBinaryTupleInt(buf *bytes.Buffer, kind int, n int64) error {
	switch kind {
	case BtInt8:
		if n < math.MinInt8 || n > math.MaxInt8 {
			return fmt.Errorf("%d does not fit into TINYINT", n)
		}
		buf.WriteByte(byte(int8(n)))
	case BtInt16:
		if n >= math.MinInt8 && n <= math.MaxInt8 {
			buf.WriteByte(byte(int8(n)))
			return nil
		}
		if n < math.MinInt16 || n > math.MaxInt16 {
			return fmt.Errorf("%d does not fit into SMALLINT", n)
		}
		putUint16(buf, uint16(int16(n)))
	case BtInt32:
		if n >= math.MinInt8 && n <= math.MaxInt8 {
			buf.WriteByte(byte(int8(n)))
			return nil
		}
		if n >= math.MinInt16 && n <= math.MaxInt16 {
			putUint16(buf, uint16(int16(n)))
			return nil
		}
		if n < math.MinInt32 || n > math.MaxInt32 {
			return fmt.Errorf("%d does not fit into INT", n)
		}
		putUint32(buf, uint32(int32(n)))
	case BtInt64:
		if n >= math.MinInt16 && n <= math.MaxInt16 {
			putUint16(buf, uint16(int16(n)))
			return nil
		}
		if n >= math.MinInt32 && n <= math.MaxInt32 {
			putUint32(buf, uint32(int32(n)))
			return nil
		}
		putUint64(buf, uint64(n))
	default:
		return fmt.Errorf("unsupported integer kind: %d", kind)
	}
	return nil
}

func putUint16(buf *bytes.Buffer, v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	buf.Write(b[:])
}

func putUint32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func putUint64(buf *bytes.Buffer, v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	buf.Write(b[:])
}

// twosComplementBytes returns the big-endian two's-complement representation of v,
// equivalent to java.math.BigInteger#toByteArray.
func twosComplementBytes(v *big.Int) []byte {
	if v == nil {
		return []byte{0}
	}
	if v.Sign() >= 0 {
		b := v.Bytes()
		if len(b) == 0 {
			return []byte{0}
		}
		if b[0]&0x80 != 0 {
			return append([]byte{0}, b...)
		}
		return b
	}

	// Negative value: take the minimal two's-complement representation, which is
	// "2^(8*L) + v" for a suitable byte length L, then trim redundant leading 0xff bytes.
	length := (v.BitLen() + 8) / 8
	twos := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(length*8)), v)
	b := twos.Bytes()
	for len(b) > 1 && b[0] == 0xff && b[1]&0x80 != 0 {
		b = b[1:]
	}
	if len(b) == 0 || b[0]&0x80 == 0 {
		// Keep the sign bit set so the value stays negative.
		b = append([]byte{0xff}, b...)
	}
	return b
}

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return v == "true" || v == "TRUE" || v == "1", nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int:
		return v != 0, nil
	case float64:
		return v != 0, nil
	}
	return false, fmt.Errorf("cannot convert %T to BOOLEAN", val)
}

func toInt64(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	}
	return 0, fmt.Errorf("cannot convert %T to integer", val)
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	}
	return 0, fmt.Errorf("cannot convert %T to float", val)
}

func toDecimal(val interface{}, scale int) (Decimal, error) {
	switch v := val.(type) {
	case Decimal:
		return v, nil
	case *Decimal:
		if v != nil {
			return *v, nil
		}
	case int:
		return Decimal{Unscaled: big.NewInt(int64(v)), Scale: scale}, nil
	case int32:
		return Decimal{Unscaled: big.NewInt(int64(v)), Scale: scale}, nil
	case int64:
		return Decimal{Unscaled: big.NewInt(v), Scale: scale}, nil
	case float64:
		return decimalFromFloat64(v, scale), nil
	case string:
		return decimalFromString(v, scale), nil
	}
	return Decimal{}, fmt.Errorf("cannot convert %T to DECIMAL", val)
}

// decimalFromString parses a decimal string and scales it to the given scale,
// truncating towards zero (the same behaviour the Ignite server applies for a DECIMAL column).
func decimalFromString(val string, scale int) Decimal {
	r, ok := new(big.Rat).SetString(val)
	if !ok {
		return Decimal{Unscaled: big.NewInt(0), Scale: scale}
	}
	mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	r.Mul(r, new(big.Rat).SetInt(mul))
	return Decimal{Unscaled: new(big.Int).Quo(r.Num(), r.Denom()), Scale: scale}
}

// decimalFromFloat64 converts a float64 to a DECIMAL with the given scale.
func decimalFromFloat64(val float64, scale int) Decimal {
	return decimalFromString(strconv.FormatFloat(val, 'f', -1, 64), scale)
}

// timeFromString parses the most common timestamp / date / time layouts.
func timeFromString(val string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05.999999999",
		"15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", val)
}

func toTime(val interface{}) (time.Time, error) {
	switch v := val.(type) {
	case time.Time:
		return v, nil
	case Timestamp:
		return v.Time, nil
	case DateTime:
		return v.Time, nil
	case Date:
		return v.Time, nil
	case Time:
		return v.Time, nil
	case string:
		return timeFromString(v)
	}
	return time.Time{}, fmt.Errorf("cannot convert %T to time", val)
}
