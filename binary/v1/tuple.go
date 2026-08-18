package ignite3

// Ref : https://cwiki.apache.org/confluence/display/IGNITE/IEP-92%3A+Binary+Tuple+Format

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"time"
)

const (
	headerSize  = 1
	tsvTypeSize = 1

	localDebug = false
)

// Un tuple est une ligne de valeurs repondants 脿 un sch茅ma. Par exemple une ligne de reponse SQL.

// Schema minimal : ordre des colonnes (+ types si n茅cessaire)
type Column struct {
	Name string
	// Type info optionnelle (string, enum, etc.)
	// Type ColumnType
}

type Schema struct {
	Columns []Column
	indices map[string]int // m_indices : name -> index dans Columns
}

func NewSchema(cols []string) *Schema {
	s := &Schema{indices: make(map[string]int)}
	s.Columns = make([]Column, len(cols))
	for i, c := range cols {
		s.Columns[i] = Column{Name: c}
		s.indices[c] = i
	}
	return s
}

type scale int16
// Temporary holder so we can have the global view and decide offset size regarding total length
// We need to keep index to handle null values which are not written
type typeScaleValue struct {
	Type  int8
	Scale scale
	Precision int
	Value []byte
}

type typeScaleValues []typeScaleValue

type tupleHeader int

// Ref modules/binary-tuple/src/main/java/org/apache/ignite/internal/binarytuple/BinaryTupleBuilder.java
type Tuple struct {
	binTupleValue []byte // Raw tuple value
	BinaryTuple   *bytes.Buffer

	values        []typeScaleValue
	nulls         []bool

	numElements   int         // Number of elements in the tuple
	entrySize     tupleHeader // Size of an offset table entry
	entryBase     int         // Position of variable-length offset table
	valueBase     int         // Starting position of variable-length values
}

func NewTuple() *Tuple {
	var t Tuple
	t.BinaryTuple = bytes.NewBuffer(t.binTupleValue)
	return &t
}

func NewTupleFromByteArrayWithMetadata(meta *ResultSetMetadata, bufr []byte) (*Tuple, error) {
	var tup Tuple
	var err error

	if len(bufr) < headerSize {
		return nil, fmt.Errorf("tuple length is too small")
	}
	tup.numElements   = len(meta.Columns)
	tup.binTupleValue = make([]byte, len(bufr))
	tup.binTupleValue = bufr

	if err = tup.entrySize.fromHeader(tup.binTupleValue[0]); err != nil {
		return nil, err
	}

	tup.entryBase = headerSize
	tup.valueBase = headerSize + (tup.numElements * int(tup.entrySize))
	if localDebug { fmt.Printf("DEBUG: tup.entryBase = %d, tup.entrySize = %d, tup.elemCount = %d, tupvalueBase = %d\n", tup.entryBase, tup.entrySize,tup.numElements, tup.valueBase) }

	if localDebug { fmt.Printf("---------------  DEBUG  ---------------\n") }
	if localDebug { fmt.Printf("DEBUG: binary tuple value is %x\n", tup.binTupleValue) }

	var tsv typeScaleValue
	var offStart, offEnd int
	for i := 0 ; i < tup.numElements ; i++ {
		tsv.Type = int8(meta.Columns[i].Type.TypeId)
		tsv.Scale = scale(meta.Columns[i].Scale)
		tsv.Precision = meta.Columns[i].Precision
		offStart = tup.entryOffset(i-1)
		if i == tup.numElements-1 {
			offEnd = len(tup.binTupleValue) - tup.valueBase
		} else {
			offEnd = tup.entryOffset(i)
			if err != nil {
				return nil, err
			}
		}
		if localDebug { fmt.Printf("DEBUG: value offset start at %x in value table, end at %x in value table, have size of %x\n", offStart, offEnd, offEnd-offStart) }

		// Now get value
		tsv.Value = make([]byte, offEnd-offStart)
		switch tsv.Type {
			// numbers are stored big-endian, all the rest is little-endian
			case typeSMALLINT, typeINT, typeBIGINT :
				for i := 0; i < offEnd-offStart; i++ {
					tsv.Value[i] = tup.binTupleValue[tup.valueBase+offEnd-(i+1)]
				}
			// Handle 0x80 Value for "some" variable length types. It means zero-length value.
			case typeVARCHAR, typeVARBINARY:
				for i := 0; i < offEnd-offStart; i++ {
					tsv.Value[i] = tup.binTupleValue[tup.valueBase+offStart+i]
				}
				if tsv.Value[0] == 0x80 {
					// 0x80 is doubled if value starts with 0x80
					if len(tsv.Value) > 1 {
						tsv.Value = tsv.Value[1:]
					} else {
						tsv.Value = []byte{}
					}
				}
			default :
				for i := 0; i < offEnd-offStart; i++ {
					tsv.Value[i] = tup.binTupleValue[tup.valueBase+offStart+i]
				}
		}
		if localDebug { fmt.Printf("DEBUG: value is %v / %x\n", tsv.Value, tsv.Value) }
		tup.values = append(tup.values, tsv)
	}

	return &tup, nil
}

// Build a tsv and append to tuple
func (t *Tuple) AddValue(v interface{}) error {
	switch v.(type) {
		case int:
			return t.setInt64(int64(v.(int)))
		case int64:
			return t.setInt64(v.(int64))
		case int32:
			return t.setInt32(v.(int32))
		case int16:
			return t.setInt16(v.(int16))
		case int8:
			return t.setInt8(v.(int8))
		case float64:
			return t.setFloat64(v.(float64))
		case float32:
			return t.setFloat32(v.(float32))
		case bool:
			return t.setBool(v.(bool))
		case byte:
			return t.setByte(v.(byte))
		case []byte:
			return t.setByteArray(v.([]byte))
		case string:
			return t.setString(v.(string))
		case Decimal:
			return t.setDecimal(v.(Decimal))
		case Uuid:
			return t.setUuid(v.(Uuid))
		case DateTime:
			return t.setDateTime(v.(DateTime))
		case Date:
			return t.setDate(v.(Date))
		case Time:
			return t.setTime(v.(Time))
		case Timestamp:
			return t.setTimestamp(v.(Timestamp))
		case nil:
			return t.setNil()
	}
	// time.Time type is unsupported, use ignite types so we can assume ignite type
	return fmt.Errorf("Unsupported type: %T", v)
}

func (t *Tuple) Bytes() ([]byte, error) {
	var err error
	// First get total length to decide on offset entry size
	total := 0
	for _, v := range t.values {
		total += v.getLength()
	}

	// Set offset entry size
	if total < math.MaxUint8 {
		t.entrySize = 1
	} else if total < math.MaxUint16 {
		t.entrySize = 2
	} else if total < math.MaxUint32 {
		t.entrySize = 4
	} else {
		t.entrySize = 8
	}

	// Write header
	var payload []byte
	payloadWriter := bytes.NewBuffer(payload)
	payloadWriter.Write([]byte{t.entrySize.toHeader()})

	// Write offset table
	prevItemLastOffset := 0

	for _, v := range t.values {
		switch t.entrySize.int() {
			case 1:
				payloadWriter.Write([]byte{byte(prevItemLastOffset+1)})
				payloadWriter.Write([]byte{byte(prevItemLastOffset+2)})
				payloadWriter.Write([]byte{byte(prevItemLastOffset+v.getLength())})
			case 2:
				binary.Write(payloadWriter, binary.LittleEndian, uint16(prevItemLastOffset+1))
				binary.Write(payloadWriter, binary.LittleEndian, uint16(prevItemLastOffset+2))
				binary.Write(payloadWriter, binary.LittleEndian, uint16(prevItemLastOffset+v.getLength()))
			case 4:
				binary.Write(payloadWriter, binary.LittleEndian, uint32(prevItemLastOffset+1))
				binary.Write(payloadWriter, binary.LittleEndian, uint32(prevItemLastOffset+2))
				binary.Write(payloadWriter, binary.LittleEndian, uint32(prevItemLastOffset+v.getLength()))
			case 8:
				binary.Write(payloadWriter, binary.LittleEndian, uint64(prevItemLastOffset+1))
				binary.Write(payloadWriter, binary.LittleEndian, uint64(prevItemLastOffset+2))
				binary.Write(payloadWriter, binary.LittleEndian, uint64(prevItemLastOffset+v.getLength()))
		}
		prevItemLastOffset += v.getLength()
	}

	// Write types, scales and values
	for _, v := range t.values {
		if err = binary.Write(payloadWriter, binary.LittleEndian, byte(v.Type)); err != nil {
			return []byte{}, err
		}
		if v.Scale.getLength() == 1 {
			if err = binary.Write(payloadWriter, binary.LittleEndian, byte(v.Scale)); err != nil {
				return []byte{}, err
			}
		} else {
			if err = binary.Write(payloadWriter, binary.LittleEndian, v.Scale); err != nil {
				return []byte{}, err
			}
		}
		if err = binary.Write(payloadWriter, binary.LittleEndian, v.Value); err != nil {
			return []byte{}, err
		}
	}

	return payloadWriter.Bytes(), nil
}

func (t *Tuple) GetLength() int {
	return t.numElements
}

func (t *Tuple) GetValue(index int, valType int) (interface{}, error) {
	if index < 0 || index > len(t.values) {
		return nil, fmt.Errorf("Index is invalid")
	}
	switch valType {
		case typeNULL:
			return nil, nil
		case typeBOOLEAN:
			if t.values[index].Value[0] == 0 {
				return false, nil
			} else {
				return true, nil
			}
		case typeTINYINT, typeSMALLINT, typeINT, typeBIGINT:
			return t.values[index].expandIntValue(valType)
		case typeREAL:
			bits := binary.LittleEndian.Uint32(t.values[index].Value)
			return math.Float32frombits(bits), nil
		case typeDOUBLE:
			return t.values[index].expandDoubleValue()
		case typeDECIMAL:
			// The scale is a signed int16 stored little-endian (it may be negative,
			// e.g. the server stores 12312300 as unscaled=123123 scale=-2).
			scale := int(int16(binary.LittleEndian.Uint16(t.values[index].Value[:2])))
			// Unscaled is a big-endian two's-complement value (IEP-92), sign-extend it
			raw := t.values[index].Value[2:]
			n := new(big.Int).SetBytes(raw)
			if len(raw) > 0 && raw[0]&0x80 != 0 {
				sub := new(big.Int).Lsh(big.NewInt(1), uint(8*len(raw)))
				n.Sub(n, sub)
			}
			return Decimal{Unscaled: n, Scale: scale}, nil
		case typeDATE:
			var date time.Time
			var err error
			if date, err = decodeDate(t.values[index].Value); err != nil {
				return nil, err
			}
			return time.Date(date.Year(),date.Month(),date.Day(),0,0,0,0,time.UTC), nil
		case typeTIME:
			var tm time.Time
			var err error
			if tm, err = decodeTime(t.values[index].Value, t.values[index].Precision); err != nil {
				return nil, err
			}
			return tm, nil
		case typeDATETIME:
			var date time.Time
			var tm time.Time
			var err error
			if len(t.values[index].Value) > 6 {
				if date, err = decodeDate(t.values[index].Value); err != nil {
					return nil, err
				}
				if tm, err = decodeTime(t.values[index].Value, t.values[index].Precision); err != nil {
					return nil, err
				}
				return time.Date(date.Year(),date.Month(),date.Day(),tm.Hour(),tm.Minute(),tm.Second(),tm.Nanosecond(),time.UTC), nil
			} else {
				// Une valeur de date non d茅finie (脿 NULL) sera rendue en "0001-01-01 00:00:00 +0000 UTC"
				return time.Time{}, nil
			}
		case typeTIMESTAMP:
			var ts time.Time
			var err error
			if ts, err = decodeTimestamp(t.values[index].Value, t.values[index].Precision); err != nil {
				return nil, err
			}
			return ts, nil
		case typeUUID:
			u := Uuid{[16]byte(t.values[index].Value)}
			u.Flip()
			return u, nil
		case typeVARCHAR:
			return string(t.values[index].Value), nil
		case typeVARBINARY:
			return []byte(t.values[index].Value), nil
		default:
			return nil, fmt.Errorf("Type not implemented : %x", valType)
	}
}


/****************************************************************
 * Internal raw functions
 ***************************************************************/
// Scale can compress to one byte
func (s scale) getLength() int {
	if s < math.MaxUint8 {
		return 1
	} else {
		return 2
	}
}

func (t tupleHeader) toHeader() byte {
	if t == 8 {
		return byte(0x3)
	}
	return byte(t >> 1)
}

func (t *tupleHeader) fromHeader(header byte) (error) {
	switch header {
		case 3:
			*t = 8
		case 0:
			*t = 1
		case 1, 2:
			*t = tupleHeader(header) << 1
		default:
			return fmt.Errorf("Invalid header: %x", header)
	}
	return nil
}

func (t tupleHeader) int() int {
	return int(t)
}

func (t *Tuple) addTsv(v typeScaleValue) {
	t.values = append(t.values, v)
}

// index is not written so we don't count
func (t *typeScaleValue) getLength() int {
	if t.Type == typeNull {
		return 0
	}
	// Scale can compress to one byte
	return tsvTypeSize + t.Scale.getLength() + len(t.Value)
}

func (t *Tuple) setNil() error {
	tsv := typeScaleValue{
		Type: typeNULL,
		Scale: 0,
	}
	// If a value is equal to NULL then it is absent in the value area. This means that in the offset table the corresponding entry is equal to the previous entry.

	t.addTsv(tsv)
	return nil
}

func (t *Tuple) setBool(b bool) error {
	tsv := typeScaleValue{
		Type:  typeBOOLEAN,
		Scale: 0,
	}
	if b {
		tsv.Value = []byte{1}
	} else {
		tsv.Value = []byte{0}
	}
	t.addTsv(tsv)
	return nil
}

func (t *Tuple) setByte(b byte) error {
	tsv := typeScaleValue{
		Type:  typeTINYINT,
		Scale: 0,
	}
	tsv.Value = []byte{b}
	t.addTsv(tsv)
	return nil
}

// VARBINARY
func (t *Tuple) setByteArray(b []byte) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeVARBINARY,
		Scale: 0,
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, b); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// VARCHAR
func (t *Tuple) setString(b string) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeVARCHAR,
		Scale: 0,
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, []byte(b)); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// TINYINT
func (t *Tuple) setInt8(v int8) error {
	t.setByte(byte(v))
	return nil
}

// SMALLINT
func (t *Tuple) setInt16(v int16) error {
	tsv := typeScaleValue{
		Type:  typeSMALLINT,
		Scale: 0,
	}
	tsv.Value = encodeIntLe(int64(v))
	t.addTsv(tsv)
	return nil
}

// INT
func (t *Tuple) setInt32(v int32) error {
	tsv := typeScaleValue{
		Type:  typeINT,
		Scale: 0,
	}
	tsv.Value = encodeIntLe(int64(v))
	t.addTsv(tsv)
	return nil
}

// BIGINT
func (t *Tuple) setInt64(v int64) error {
	tsv := typeScaleValue{
		Type:  typeBIGINT,
		Scale: 0,
	}
	tsv.Value = encodeIntLe(v)
	t.addTsv(tsv)
	return nil
}

// REAL
func (t *Tuple) setFloat32(v float32) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeREAL,
		Scale: 0,
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, v); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// DOUBLE
func (t *Tuple) setFloat64(v float64) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeDOUBLE,
		Scale: 0,
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, v); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// DECIMAL
func (t *Tuple) setDecimal(v Decimal) error {
	var err error
	/*value = BinaryTupleCommon.shrinkDecimal(value, scale);*/
	// See CatalogUtils.MAX_DECIMAL_SCALE = Short.MAX_VALUE
	if (v.Scale > math.MaxInt16) {
		return fmt.Errorf("Decimal scale is too large")
	}
	if (v.Scale < math.MinInt16) {
		return fmt.Errorf("Decimal scale is too small")
	}
	tsv := typeScaleValue{
		Type:  typeDECIMAL,
		Scale: scale(v.Scale),
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, (int16(v.Scale))); err != nil {
		return err
	}
	tsv.Value = append(tsv.Value, encodeDecimalUnscaled(v.Unscaled)...)
	t.addTsv(tsv)
	return nil
}

// UUID
func (t *Tuple) setUuid(v Uuid) error {
	var err error
	/*value = BinaryTupleCommon.shrinkDecimal(value, scale);*/
	// See CatalogUtils.MAX_DECIMAL_SCALE = Short.MAX_VALUE
	tsv := typeScaleValue{
		Type:  typeUUID,
		Scale: 0,
	}
	// Flip it to write on wire
	v.Flip()
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, v); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// DATETIME (aka TIMESTAMP)
func (t *Tuple) setDateTime(v DateTime) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeDATETIME,
		Scale: 0,
	}
	var dat, tim []byte
	if dat, err = encodeDate(v.Time.Year(), int(v.Time.Month()), v.Time.Day()); err != nil {
		return err
	}
	if tim, err = encodeTime(v.Time.Hour(), v.Time.Minute(), v.Time.Second(), v.Time.Nanosecond()); err != nil {
		return err
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, append(dat, tim...)); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// TIMESTAMP (aka TIMESTAMP WITH LOCAL TIME ZONE)
func (t *Tuple) setTimestamp(v Timestamp) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeTIMESTAMP,
		Scale: 0,
	}
	ts := v.Time.Unix()
	nsec := int32(v.Time.Nanosecond())
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, ts); err != nil {
		return err
	}
	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, nsec); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// DATE
func (t *Tuple) setDate(v Date) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeDATE,
		Scale: 0,
	}
	var dat []byte
	if dat, err = encodeDate(v.Time.Year(), int(v.Time.Month()), v.Time.Day()); err != nil {
		return err
	}

	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, dat); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

// TIME
func (t *Tuple) setTime(v Time) error {
	var err error
	tsv := typeScaleValue{
		Type:  typeTIME,
		Scale: 0,
	}
	var tim []byte
	if tim, err = encodeTime(v.Time.Hour(), v.Time.Minute(), v.Time.Second(), v.Time.Nanosecond()); err != nil {
		return err
	}

	if tsv.Value, err = binary.Append(tsv.Value, binary.LittleEndian, tim); err != nil {
		return err
	}
	t.addTsv(tsv)
	return nil
}

func (t *Tuple) entryOffset(i int) int {
	if i < 0 {
		return 0
	}
	pos := t.entryBase + i*int(t.entrySize)
	if localDebug { fmt.Printf("DEBUG: pos = %x, t.entrySize = %x\n", pos, t.entrySize) }
	switch t.entrySize {
		case 1:
			return int(t.binTupleValue[pos])
		case 2:
			return int(binary.LittleEndian.Uint16(t.binTupleValue[pos : pos+2]))
		case 4:
			return int(binary.LittleEndian.Uint32(t.binTupleValue[pos : pos+4]))
		case 8:
			return int(binary.LittleEndian.Uint64(t.binTupleValue[pos : pos+8]))
	}
	return -1
}

// encodeIntLe encodes an int64 as a little-endian two's-complement byte sequence
// with width 1, 2, 4 or 8 bytes. The Ignite 3 server reads integer tuple elements
// as little-endian signed values and only accepts the standard fixed widths.
func encodeIntLe(v int64) []byte {
	switch {
	case v >= math.MinInt8 && v <= math.MaxInt8:
		return []byte{byte(v)}
	case v >= math.MinInt16 && v <= math.MaxInt16:
		return []byte{byte(v), byte(v >> 8)}
	case v >= math.MinInt32 && v <= math.MaxInt32:
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	default:
		return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24), byte(v >> 32), byte(v >> 40), byte(v >> 48), byte(v >> 56)}
	}
}


// decodeBigEndianTwosComplement decodes a big-endian two's-complement byte sequence (IEP-92).
func decodeBigEndianTwosComplement(raw []byte) (int64, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	if len(raw) > 8 {
		return 0, fmt.Errorf("invalid integer length: %d", len(raw))
	}
	var v uint64
	for _, b := range raw {
		v = v<<8 | uint64(b)
	}
	if raw[0]&0x80 != 0 {
		v -= uint64(1) << uint(8*len(raw))
	}
	return int64(v), nil
}

// encodeDecimalUnscaled encodes a big.Int as a big-endian two's-complement byte
// sequence (IEP-92 decimal unscaled representation).
// Unlike big.Int.Bytes() (unsigned magnitude, empty for zero), this yields a
// signed minimal-length sequence the server interprets correctly.
func encodeDecimalUnscaled(n *big.Int) []byte {
	switch n.Sign() {
	case 0:
		return []byte{0}
	case 1:
		b := n.Bytes()
		if b[0]&0x80 != 0 {
			b = append([]byte{0x00}, b...)
		}
		return b
	default:
		abs := new(big.Int).Neg(n)
		bits := (abs.BitLen() + 7) / 8
		if bits == 0 {
			bits = 1
		}
		mod := new(big.Int).Lsh(big.NewInt(1), uint(8*bits))
		twos := new(big.Int).Sub(mod, abs)
		b := twos.Bytes()
		if b[0]&0x80 == 0 {
			b = append([]byte{0xFF}, b...)
		}
		return b
	}
}

// Values are read compressed, so we need to expand them to fit their real type
// From IEP-92 : A Number field is represented as a variable-length byte sequence containing
// the two's-complement binary value. Unlike all the other types here the bytes go in big-endian order.
func (tsv *typeScaleValue) expandIntValue(valType int) (interface{}, error) {
	v, err := decodeBigEndianTwosComplement(tsv.Value)
	if err != nil {
		return nil, err
	}
	switch valType {
		case typeTINYINT:
			return int8(v), nil
		case typeSMALLINT:
			return int16(v), nil
		case typeINT:
			return int32(v), nil
		case typeBIGINT:
			return v, nil
		default:
			return nil, fmt.Errorf("Type not implemented: %x", valType)
	}
}

func (tsv *typeScaleValue) expandDoubleValue() (interface{}, error) {
	switch len(tsv.Value) {
		case 4:
			return math.Float32frombits(binary.LittleEndian.Uint32(tsv.Value)), nil
		case 8:
			return math.Float64frombits(binary.LittleEndian.Uint64(tsv.Value)), nil
		default:
			return nil, fmt.Errorf("Length not implemented for DOUBLE: %d", len(tsv.Value))
	}
}

func encodeDate(year int, month int, day int) ([]byte, error) {
	// Validate ranges (spec does not restrict to Gregorian validity, but we do basic bounds).
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("Month out of range")
	}
	if day < 1 || day > 31 {
		return nil, fmt.Errorf("Day out of range")
	}
	// Year is 15-bit signed two's complement: range [-16384, 16383]
	if year < -16384 || year > 16383 {
		return nil, fmt.Errorf("Year out of 15-bit signed range")
	}

	var y uint32
	if year < 0 {
		y = uint32((1 << 15) + year) // two's complement in 15 bits
	} else {
		y = uint32(year)
	}

	v := (uint32(day) & 0x1F) | ((uint32(month) & 0x0F) << 5) | ((y & 0x7FFF) << 9)
	out := make([]byte, 3)
	out[0] = byte(v & 0xFF)
	out[1] = byte((v >> 8) & 0xFF)
	out[2] = byte((v >> 16) & 0xFF)
	return out, nil
}

func encodeTime(hour, minute, second, nsec int) ([]byte, error) {
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("Hour out of range")
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("Minute out of range")
	}
	if second < 0 || second > 59 {
		return nil, fmt.Errorf("Second out of range")
	}
	if nsec < 0 || nsec > 999_999_999 {
		return nil, fmt.Errorf("Nanosec out of range")
	}

	fracVal := uint64(nsec)
	// Pack low bits = fraction, then seconds, minutes, hours, then pad at the top.
	var v uint64
	shift := uint(0)
	// fractional part
	v |= fracVal << shift
	shift += 30
	// seconds (6 bits)
	v |= (uint64(second) & 0x3F) << shift
	shift += 6
	// minutes (6 bits)
	v |= (uint64(minute) & 0x3F) << shift
	shift += 6
	// hours (5 bits)
	v |= (uint64(hour) & 0x1F) << shift
	shift += 5

	out := make([]byte, 48/8)
	for i := range out {
		out[i] = byte((v >> (8 * uint(i))) & 0xFF) // little-endian bytes
	}
	return out, nil
}

func decodeDate(raw []byte) (time.Time, error) {
	var tm time.Time
	raw4 := make([]byte, 4)
	for i := 0 ; i < 3 ; i++ {
		raw4[i] = raw[i]
	}
	day  := binary.LittleEndian.Uint32(raw4) & 0b11111
	mon  := binary.LittleEndian.Uint32(raw4) >> 5 & 0b1111
	year := binary.LittleEndian.Uint32(raw4) >> 9 & 0b111111111111111
	// FIXME : Location
	tm = time.Date(int(year), time.Month(mon), int(day), 0, 0, 0, 0, time.UTC)
	return tm, nil
}

func decodeTime(raw []byte, precision int) (time.Time, error) {
	var tm time.Time

	if precision == 6 || precision == 0 {
		// Time is in milliseconds
		raw4 := make([]byte, 4)
		for i := len(raw)-1 ; i >= len(raw)-4 ; i-- {
			raw4[4-(len(raw)-i)] = raw[i]
		}
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint32(raw4) & 0b1111111111)   * time.Millisecond)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint32(raw4) >> 10 & 0b111111) * time.Second)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint32(raw4) >> 16 & 0b111111) * time.Minute)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint32(raw4) >> 22 & 0b11111)  * time.Hour)
		return tm, nil
	} else if precision == 9 {
		// Time is in microseconds
		raw8 := make([]byte, 8)
		for i := len(raw)-1 ; i >= len(raw)-5 ; i-- {
			raw8[5-(len(raw)-i)] = raw[i]
		}
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) & 0b11111111111111111111) * time.Microsecond)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 20 & 0b111111) * time.Second)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 26 & 0b111111) * time.Minute)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 32 & 0b11111)  * time.Hour)
		return tm, nil
	} else if precision == 12 {
		// Time is in nanosecond
		raw8 := make([]byte, 8)
		for i := len(raw)-1 ; i >= len(raw)-6 ; i-- {
			raw8[6-(len(raw)-i)] = raw[i]
		}
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) & 0b111111111111111111111111111111) * time.Microsecond)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 30 & 0b111111) * time.Second)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 36 & 0b111111) * time.Minute)
		tm = tm.Add(time.Duration(binary.LittleEndian.Uint64(raw8) >> 42 & 0b11111)  * time.Hour)
		return tm, nil
	}

	return tm, fmt.Errorf("Precision not supported: %d", precision)
}

func decodeTimestamp(raw []byte, precision int) (time.Time, error) {
	ts := binary.LittleEndian.Uint64(raw)
	nsec := binary.LittleEndian.Uint32(raw[8:])
	return time.Unix(int64(ts), int64(nsec)), nil
}
