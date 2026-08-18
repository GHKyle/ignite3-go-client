package ignite3

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"io"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
	//"github.com/GHKyle/ignite3-go-client/binary/errors"
)

const (
	// Duplicate with ColumnTypes in client-result-set.go
	typeNull,      typeNULL      =  0,  0
	typeBool,      typeBOOLEAN   =  1,  1
	typeByte,      typeTINYINT   =  2,  2
	typeShort,     typeSMALLINT  =  3,  3
	typeInt,       typeINT       =  4,  4
	typeLong,      typeBIGINT    =  5,  5
	typeFloat,     typeREAL      =  6,  6
	typeDouble,    typeDOUBLE    =  7,  7
	typeDecimal,   typeDECIMAL   =  8,  8
	typeDate,      typeDATE      =  9,  9
	typeTime,      typeTIME      = 10, 10
	typeDateTime,  typeDATETIME  = 11, 11  // aka TIMESTAMP
	typeTimestamp, typeTIMESTAMP = 12, 12  // aka TIMESTAMP WITH LOCAL TIME ZONE
	typeUuid,      typeUUID      = 13, 13
	// unused
	typeString,    typeVARCHAR   = 15, 15
        typeByteArray, typeVARBINARY = 16, 16
)

var MAGIC_BYTES = []byte{0x49, 0x47, 0x4E, 0x49}

// Char is Apache Ignite "char" type
type Char rune

type Uuid struct {
	uuid.UUID
}

type Decimal struct {
	Unscaled *big.Int // number without dot, Ex.: 31415926535
	Scale    int      // number of decimal places, Ex.: 10. So with Unscaled, we got 3.1415926535
}

// Ignite type to use in sql driver
type Date struct { Time time.Time }
type Time struct { Time time.Time }
type DateTime struct { Time time.Time }
type Timestamp struct { Time time.Time }


func (d Decimal) BigRat() *big.Rat {
	if d.Scale < 0 {
		// Negative scale: value = Unscaled * 10^|Scale| (e.g. Unscaled=123123, Scale=-2 => 12312300)
		pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-d.Scale)), nil)
		return new(big.Rat).SetInt(new(big.Int).Mul(d.Unscaled, pow))
	}
	// Positive scale: value = Unscaled / 10^Scale
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.Scale)), nil)
	return new(big.Rat).SetFrac(d.Unscaled, pow)
}

func (d Decimal) String() string {
	if d.Scale < 0 {
		// value = Unscaled * 10^|Scale|, so just append trailing zeros
		return d.Unscaled.String() + strings.Repeat("0", -d.Scale)
	}
	return d.BigRat().FloatString(d.Scale)
}

func (u *Uuid) FromUUID(uid uuid.UUID) {
	u.UUID = uid
}

func (u *Uuid) FromString(uid string) {
	u.UUID, _ = uuid.Parse(uid)
}

func (u *Uuid) New() {
	u.UUID = uuid.New()
}

func (u *Uuid) Flip() {
	for i := 3; i >= 0; i-- {
		opp := 7 - i
		u.UUID[i], u.UUID[opp] = u.UUID[opp], u.UUID[i]
	}
	for i := 3; i >= 0; i-- {
		opp := 15 - i
		u.UUID[i+8], u.UUID[opp] = u.UUID[opp], u.UUID[i+8]
	}
}

func (u *Uuid) MarshalMsgpack() ([]byte, error) {
	buf := new(bytes.Buffer)
	u.Flip()
	if err := binary.Write(buf, binary.BigEndian, u); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

// Prend en entree les donnees brutes, sans l'entete msgpack
func (u *Uuid) UnmarshalMsgpack(b []byte) (error) {
	buf := bytes.NewReader(b)

	_, err := io.ReadFull(buf, u.UUID[0:8])
	if err != nil {
		return err
	}
	_, err = io.ReadFull(buf, u.UUID[8:16])
	if err != nil {
		return err
	}
	u.Flip()
	return nil
}

func (d *DateTime) FromTime(t time.Time) {
	d.Time = t
}

func (ts *Timestamp) FromTime(t time.Time) {
	ts.Time = t
}

//const tsLayout = "2006-01-02 15:04:05.999999"
const tsLayout = "2006-01-02 15:04:05.999999999 -0700 MST"

func (t *Timestamp) Scan(src any) error {
	switch v := src.(type) {
		case time.Time:
			t.Time = v
			return nil
		case []byte:
			tt, err := time.Parse(tsLayout, string(v))
			if err != nil {
				return fmt.Errorf("Timestamp.Scan: parse from []byte: %w", err)
			}
			t.Time = tt
			return nil
		case string:
			tt, err := time.Parse(tsLayout, v)
			if err != nil {
				return fmt.Errorf("Timestamp.Scan: parse from string: %w", err)
			}
			t.Time = tt
			return nil
		case nil:
			t.Time = time.Time{}
			return nil
		default:
			return fmt.Errorf("Timestamp.Scan: unsupported type %T", src)
	}
}


func (d *Date) FromTime(t time.Time) {
	d.Time = t
}

func (d *Time) FromTime(t time.Time) {
	d.Time = t
}

// Register Msgpack extensions
func init() {
	msgpack.RegisterExt(3, (*Uuid)(nil))
}

// v3_WIP : NOK quand il y a un msg d'erreur (a provoquer en mettant une version incompatible au handshake)
func TryReadError(r io.Reader) (interface{}, error) {
	dec := msgpack.NewDecoder(r)
	return dec.DecodeInterface()
}

// Try to read Nil. If Nil, reader is advanced to next message. If not nil, reader is not advanced
func TryReadPackedNil(r io.Reader) (bool, error) {
	dec := msgpack.NewDecoder(r)
	code, err := dec.PeekCode()
	if err != nil {
		return false, err
	}
	if code == msgpcode.Nil {
		// Advance to next value
		dec.Skip()
		return true, nil
	} else {
		return false, nil
	}
}

// Try to read Int. If Int found, reader is advanced to next message. If not Int, reader is not advanced and defaultValue is returned
func TryReadPackedIntWithDefault(r io.Reader, defaultValue int64) (int64, error) {
	dec := msgpack.NewDecoder(r)
	code, err := dec.PeekCode()
	if err != nil {
		return 0, err
	}
	if msgpcode.IsFixedNum(code) {
		dec.Skip()
		return int64(code), nil
	}
	if code == msgpcode.Int8 || code == msgpcode.Int16 || code == msgpcode.Int32 || code == msgpcode.Int64 {
		// Get value
		return dec.DecodeInt64()
	} else {
		return defaultValue, nil
	}
}

// ReadRawArrayBytes reads len bytes array value
func ReadRawByteArray(r io.Reader, len int) ([]byte, error) {
	b := make([]byte, len)
	if len == 0 {
		return b, nil
	}
	err := binary.Read(r, binary.BigEndian, &b)
	return b, err
}

// ReadBool reads "bool" value
func ReadPackedBool(r io.Reader) (bool, error) {
	dec := msgpack.NewDecoder(r)
	return dec.DecodeBool()
}

func ReadPackedInt8(r io.Reader) (int8, error) {
	dec := msgpack.NewDecoder(r)

	var i int8
	if err := dec.Decode(&i); err != nil {
		return 0, err
	}
	return i, nil
}

func ReadPackedUint8(r io.Reader) (uint8, error) {
	dec := msgpack.NewDecoder(r)

	var i uint8
	if err := dec.Decode(&i); err != nil {
		return 0, err
	}
	return i, nil
}

func ReadPackedByte(r io.Reader) (byte, error) {
	dec := msgpack.NewDecoder(r)

	var b byte
	if err := dec.Decode(&b); err != nil {
		return b, err
	}
	return b, nil
}

func ReadPackedBytes(r io.Reader) ([]byte, error) {
	dec := msgpack.NewDecoder(r)

	var b []byte
	if err := dec.Decode(&b); err != nil {
		return b, err
	}
	return b, nil
}

func ReadPackedInt16(r io.Reader) (int16, error) {
	dec := msgpack.NewDecoder(r)

	var i int16
	if err := dec.Decode(&i); err != nil {
		return i, err
	}
	return i, nil
}

func ReadPackedInt32(r io.Reader) (int32, error) {
	d := msgpack.NewDecoder(r)
	return d.DecodeInt32()
}

func ReadPackedInt(r io.Reader) (int64, error) {
	return ReadPackedInt64(r)
}

func ReadPackedInt64(r io.Reader) (int64, error) {
	d := msgpack.NewDecoder(r)
	return d.DecodeInt64()
}

func ReadPackedInt64Array(r io.Reader) ([]int64, error) {
	d := msgpack.NewDecoder(r)

	res, err := d.DecodeSlice()
	if err != nil {
		return []int64{}, err
	}
	var res64 []int64
	for _, i := range res {
		res64 = append(res64, i.(int64))
	}
	return res64, err
}

func ReadPackedString(r io.Reader) (string, error) {
	d := msgpack.NewDecoder(r)
	return d.DecodeString()
}

// ReadUUID reads "UUID" object value
func ReadPackedUUID(r io.Reader) (Uuid, error) {
	var u Uuid
	buf := make([]byte, 18)
	if err := binary.Read(r, binary.BigEndian, &buf); err != nil {
		return u, err
	}
	if err := msgpack.Unmarshal(buf, &u); err != nil {
		return u, err
	}
	return u, nil
}

func ReadPackedInt64String(r io.Reader) (int64, string, error) {
	d := msgpack.NewDecoder(r)
	k, err :=  d.DecodeInt64()
	if err != nil {
		return 0, "", err
	}
	v, err := d.DecodeString()
	if err != nil {
		return k, "", err
	}
	return k, v, nil
}

// ReadTimestamp reads "Timestamp" object value
func ReadPackedTimestamp(r io.Reader) (time.Time, error) {
	raw, err := ReadPackedInt64(r)
	if err != nil {
		return time.Time{}, err
	}
	low := raw & 0xFF % 1000 * int64(time.Millisecond)
	high := int64(raw) >> 16 / 1000
	return time.Unix(high, int64(low)).UTC(), nil
}

// WriteByte writes "byte" value
func WriteRawByte(w io.Writer, v byte) error {
	return binary.Write(w, binary.BigEndian, v)
}

// WriteBytes writes byte slice raw, not encoded
func WriteRawBytes(w io.Writer, v []byte) error {
	return binary.Write(w, binary.BigEndian, v)
}

func WriteRawInt8(w io.Writer, v int8) error {
	return WriteRawByte(w, byte(v))
}

// Write 32 bit raw (no msgpack), typically used for sending payload length
func WriteRawInt32(w io.Writer, v int32) error {
	return binary.Write(w, binary.BigEndian, v)
}

func WritePackedNull(w io.Writer) error {
	e := msgpack.NewEncoder(w)
	return e.EncodeNil()
}

func WritePackedByte(w io.Writer, v byte) error {
	return WriteRawByte(w, v)
}

func WritePackedBytes(w io.Writer, v []byte) error {
	e := msgpack.NewEncoder(w)
	return e.EncodeBytes(v)
}

func WritePackedInt64(w io.Writer, v int64) error {
	e := msgpack.NewEncoder(w)
	return e.EncodeInt(v)
}

// WriteString writes "string" value
func WritePackedString(w io.Writer, v string) error {
	e := msgpack.NewEncoder(w)
	return e.EncodeString(v)
}

// WriteOUUID writes "UUID" object value as msgpack ext 3
func WritePackedUUID(w io.Writer, v Uuid) error {
	b, err := msgpack.Marshal(&v)
	if err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, b)
}

func WritePackedTimestamp(w io.Writer, ts time.Time) error {
	ms := ts.UnixMilli()
	return WritePackedInt64(w, ms*256*256)
}

// WriteMap writes map type code
func WritePackedMap(w io.Writer, code map[string]string) error {
	WriteRawInt8(w, int8(len(code)))
	// FIXME: map browsing is not ordered, so test fail sometimes
	keys := make([]string, 0, len(code))
	for k := range code{
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for k, _ := range code {
		bcode, err := msgpack.Marshal(k)
		if err != nil {
			return err
		}
		err = WriteRawBytes(w, bcode)
		if err != nil {
			return err
		}
		bcode, err = msgpack.Marshal(code[k])
		if err != nil {
			return err
		}
		err = WriteRawBytes(w, bcode)
		if err != nil {
			return err
		}
	}
	return nil
}
