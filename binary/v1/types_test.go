package ignite3

import (
	"bytes"
	//"io"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// v3_OK
func TestWritePackedByte(t *testing.T) {
	type args struct {
		v byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: 123,
			},
			want: []byte{123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedByte(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedByte() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedByte() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

// v3_OK
func TestWriteRawByte(t *testing.T) {
	type args struct {
		v byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: 123,
			},
			want: []byte{123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WriteRawByte(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WriteRawByte() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WriteRawByte() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

// v3_OK
func TestWriteRawBytes(t *testing.T) {
	type args struct {
		v []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: []byte{0xaa,0x61,0x75,0x74,0x68,0x6e,0x2d,0x74,0x79,0x70,0x65},
			},
			want: []byte{0xaa,0x61,0x75,0x74,0x68,0x6e,0x2d,0x74,0x79,0x70,0x65},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WriteRawBytes(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WriteRawBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WriteRawBytes() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

// v3_OK
func TestWritePackedBytes(t *testing.T) {
	type args struct {
		v []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: []byte{0x04},
			},
			want: []byte{0xc4,0x01,0x04},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedBytes(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedBytes() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

func TestWritePackedInt64(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: 1234567890,
			},
			want: []byte{0xce, 0x49, 0x96, 0x02, 0xd2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedInt64(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedInt64() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedInt64() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

func TestWritePackedString(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: "test string",
			},
			want: []byte{0xab, 0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67},
		},
		{
			name: "2",
			args: args{
				v: "",
			},
			want: []byte{0xa0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedString(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedString() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

func TestWritePackedUUID(t *testing.T) {
	u, _ := uuid.Parse("d6589da7-f8b1-4687-b5bd-2ddc7362a4a4")
	var v Uuid
	v.UUID = u

	type args struct {
		v Uuid
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: v,
			},
			/*want: []byte{10, 0x87, 0x46, 0xb1, 0xf8, 0xa7, 0x9d, 0x58, 0xd6, 0xa4,
				0xa4, 0x62, 0x73, 0xdc, 0x2d, 0xbd, 0xb5},*/
			want: []byte{0xd8, 0x03, 0x87, 0x46, 0xb1, 0xf8, 0xa7, 0x9d, 0x58, 0xd6,
				0xa4, 0xa4, 0x62, 0x73, 0xdc, 0x2d, 0xbd, 0xb5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedUUID(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedUUID() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

func TestWritePackedTimestamp(t *testing.T) {
	tm := time.Date(2018, 4, 3, 14, 25, 32, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)

	type args struct {
		v time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				v: tm,
			},
			//want: []byte{33, 0xdb, 0xb, 0xe6, 0x8b, 0x62, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0},
			want: []byte{0xcf, 0x01, 0x62, 0x8b, 0xe6, 0x0b, 0xdb, 0x00, 0x00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedTimestamp(w, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedTimestamp() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedTimestamp() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

func TestWritePackedNull(t *testing.T) {
	tests := []struct {
		name    string
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			want: []byte{0xc0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := WritePackedNull(w); (err != nil) != tt.wantErr {
				t.Errorf("WritePackedNull() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(w.Bytes(), tt.want) {
				t.Errorf("WritePackedNull() = %#v, want %#v", w.Bytes(), tt.want)
			}
		})
	}
}

/*
func Test_response_ReadByte(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    byte
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{123}),
			want: 123,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadByte(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadByte() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadByte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadShort(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    int16
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0x39, 0x30}),
			want: 12345,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadShort(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadShort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadShort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadInt(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    int32
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0xD2, 0x02, 0x96, 0x49}),
			want: 1234567890,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadInt(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadLong(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    int64
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0x15, 0x81, 0xE9, 0x7D, 0xF4, 0x10, 0x22, 0x11}),
			want: 1234567890123456789,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadLong(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadLong() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadLong() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadFloat(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    float32
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0x65, 0x20, 0xf1, 0x47}),
			want: 123456.789,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadFloat(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadDouble(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    float64
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0xad, 0x69, 0x7e, 0x54, 0x34, 0x6f, 0x9d, 0x41}),
			want: 123456789.12345,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadDouble(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadDouble() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadDouble() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadChar(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    Char
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0x41, 0x0}),
			want: Char('A'),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadChar(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadChar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadBool(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    bool
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{1}),
			want: true,
		},
		{
			name: "2",
			r:    bytes.NewBuffer([]byte{0}),
			want: false,
		},
		{
			name:    "3",
			r:       bytes.NewBuffer([]byte{2}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadBool(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadOString(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    string
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{9, 0x0B, 0, 0, 0, 0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67}),
			want: "test string",
		},
		{
			name: "2",
			r: bytes.NewBuffer(
				[]byte{9, 0, 0, 0, 0}),
			want: "",
		},
		{
			name: "3",
			r: bytes.NewBuffer(
				[]byte{101}),
		},
		{
			name: "4",
			r: bytes.NewBuffer(
				[]byte{0, 0x0B, 0, 0, 0, 0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadOString(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadOString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("response.ReadOString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadUUID(t *testing.T) {
	v, _ := uuid.Parse("d6589da7-f8b1-4687-b5bd-2ddc7362a4a4")

	tests := []struct {
		name    string
		r       io.Reader
		want    uuid.UUID
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{0x87, 0x46, 0xb1, 0xf8, 0xa7, 0x9d, 0x58, 0xd6, 0xa4,
					0xa4, 0x62, 0x73, 0xdc, 0x2d, 0xbd, 0xb5}),
			want: v,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadUUID(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadUUID() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadDate(t *testing.T) {
	dm := time.Date(2018, 4, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    time.Time
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0x0, 0xa0, 0xcd, 0x88, 0x62, 0x1, 0x0, 0x0}),
			want: dm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadDate(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayBytes(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []byte
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{3, 0, 0, 0, 1, 2, 3}),
			want: []byte{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// FIXME : 3 est random
			got, err := ReadArrayBytes(tt.r, 3)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayBytes() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayShorts(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []int16
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{3, 0, 0, 0, 1, 0, 2, 0, 3, 0}),
			want: []int16{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayShorts(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayShorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayShorts() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayInts(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []int32
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{3, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}),
			want: []int32{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayInts(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayInts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayInts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayLongs(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []int64
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{3, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0}),
			want: []int64{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayLongs(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayLongs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayLongs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayFloats(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []float32
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{3, 0x0, 0x0, 0x0, 0xcd, 0xcc, 0x8c, 0x3f, 0xcd, 0xcc, 0xc, 0x40, 0x33, 0x33, 0x53, 0x40}),
			want: []float32{1.1, 2.2, 3.3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayFloats(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayFloats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayFloats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayDoubles(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []float64
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{3, 0, 0, 0, 0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0xf1, 0x3f, 0x9a, 0x99,
					0x99, 0x99, 0x99, 0x99, 0x1, 0x40, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0xa, 0x40}),
			want: []float64{1.1, 2.2, 3.3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayDoubles(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayDoubles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayDoubles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayChars(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []Char
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{3, 0, 0, 0, 0x41, 0x0, 0x42, 0x0, 0x2f, 0x4}),
			want: []Char{'A', 'B', 'Я'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayChars(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayChars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayChars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayBools(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []bool
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{3, 0, 0, 0, 1, 0, 1}),
			want: []bool{true, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayBools(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayBools() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayBools() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayOStrings(t *testing.T) {
	tests := []struct {
		name    string
		r       io.Reader
		want    []string
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer([]byte{3, 0, 0, 0,
				0x9, 3, 0, 0, 0, 0x6f, 0x6e, 0x65,
				0x9, 3, 0, 0, 0, 0x74, 0x77, 0x6f,
				0x9, 6, 0, 0, 0, 0xd1, 0x82, 0xd1, 0x80, 0xd0, 0xb8}),
			want: []string{"one", "two", "три"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayOStrings(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayOStrings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayOStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayOUUIDs(t *testing.T) {
	uid1, _ := uuid.Parse("a0c07c4c-7e2e-43d3-8eda-176881477c81")
	uid2, _ := uuid.Parse("4015b55f-72f0-48a4-8d01-64168d50f627")
	uid3, _ := uuid.Parse("827d1bf0-c5d4-4443-8708-d8b5de31fe74")

	tests := []struct {
		name    string
		r       io.Reader
		want    []uuid.UUID
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer([]byte{3, 0, 0, 0,
				10, 0xd3, 0x43, 0x2e, 0x7e, 0x4c, 0x7c, 0xc0, 0xa0, 0x81, 0x7c, 0x47, 0x81, 0x68, 0x17, 0xda, 0x8e,
				10, 0xa4, 0x48, 0xf0, 0x72, 0x5f, 0xb5, 0x15, 0x40, 0x27, 0xf6, 0x50, 0x8d, 0x16, 0x64, 0x1, 0x8d,
				10, 0x43, 0x44, 0xd4, 0xc5, 0xf0, 0x1b, 0x7d, 0x82, 0x74, 0xfe, 0x31, 0xde, 0xb5, 0xd8, 0x8, 0x87}),
			want: []uuid.UUID{uid1, uid2, uid3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayOUUIDs(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayOUUIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayOUUIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayODates(t *testing.T) {
	dm1 := time.Date(2018, 4, 3, 0, 0, 0, 0, time.UTC)
	dm2 := time.Date(2019, 5, 4, 0, 0, 0, 0, time.UTC)
	dm3 := time.Date(2020, 6, 5, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    []time.Time
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer([]byte{3, 0, 0, 0,
				11, 0x0, 0xa0, 0xcd, 0x88, 0x62, 0x1, 0x0, 0x0,
				11, 0x0, 0xf0, 0x23, 0x80, 0x6a, 0x1, 0x0, 0x0,
				11, 0x0, 0xf8, 0xc6, 0x81, 0x72, 0x1, 0x0, 0x0}),
			want: []time.Time{dm1, dm2, dm3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayODates(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayODates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayODates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadTimestamp(t *testing.T) {
	tm := time.Date(2018, 4, 3, 14, 25, 32, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    time.Time
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer(
				[]byte{0xdb, 0xb, 0xe6, 0x8b, 0x62, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0}),
			want: tm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadTimestamp(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadTimestamp() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayOTimestamps(t *testing.T) {
	tm1 := time.Date(2018, 4, 3, 14, 25, 32, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm2 := time.Date(2019, 5, 4, 15, 26, 33, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm3 := time.Date(2020, 6, 5, 16, 27, 34, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    []time.Time
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer([]byte{3, 0, 0, 0,
				33, 0xdb, 0xb, 0xe6, 0x8b, 0x62, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0,
				33, 0xa3, 0x38, 0x74, 0x83, 0x6a, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0,
				33, 0x6b, 0x1d, 0x4f, 0x85, 0x72, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0}),
			want: []time.Time{tm1, tm2, tm3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayOTimestamps(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayOTimestamps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayOTimestamps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadTime(t *testing.T) {
	tm := time.Date(1, 1, 1, 14, 25, 32, int(time.Millisecond*123), time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    time.Time
		wantErr bool
	}{
		{
			name: "1",
			r:    bytes.NewBuffer([]byte{0xdb, 0x6b, 0x18, 0x3, 0x0, 0x0, 0x0, 0x0}),
			want: tm,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadTime(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadTime() = %s, want %s", got.String(), tt.want.String())
				// t.Errorf("response.ReadTime() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadArrayOTimes(t *testing.T) {
	tm4 := time.Date(1, 1, 1, 14, 25, 32, int(time.Millisecond*123), time.UTC)
	tm5 := time.Date(1, 1, 1, 15, 26, 33, int(time.Millisecond*123), time.UTC)
	tm6 := time.Date(1, 1, 1, 16, 27, 34, int(time.Millisecond*123), time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    []time.Time
		wantErr bool
	}{
		{
			name: "1",
			r: bytes.NewBuffer([]byte{3, 0, 0, 0,
				36, 0xdb, 0x6b, 0x18, 0x3, 0x0, 0x0, 0x0, 0x0,
				36, 0xa3, 0x48, 0x50, 0x3, 0x0, 0x0, 0x0, 0x0,
				36, 0x6b, 0x25, 0x88, 0x3, 0x0, 0x0, 0x0, 0x0}),
			want: []time.Time{tm4, tm5, tm6},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadArrayOTimes(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadArrayOTimes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadArrayOTimes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_response_ReadObject(t *testing.T) {
	uid, _ := uuid.Parse("d6589da7-f8b1-4687-b5bd-2ddc7362a4a4")
	dm := time.Date(2018, 4, 3, 0, 0, 0, 0, time.UTC)
	uid1, _ := uuid.Parse("a0c07c4c-7e2e-43d3-8eda-176881477c81")
	uid2, _ := uuid.Parse("4015b55f-72f0-48a4-8d01-64168d50f627")
	uid3, _ := uuid.Parse("827d1bf0-c5d4-4443-8708-d8b5de31fe74")
	dm1 := time.Date(2018, 4, 3, 0, 0, 0, 0, time.UTC)
	dm2 := time.Date(2019, 5, 4, 0, 0, 0, 0, time.UTC)
	dm3 := time.Date(2020, 6, 5, 0, 0, 0, 0, time.UTC)
	tm := time.Date(2018, 4, 3, 14, 25, 32, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm1 := time.Date(2018, 4, 3, 14, 25, 32, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm2 := time.Date(2019, 5, 4, 15, 26, 33, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm3 := time.Date(2020, 6, 5, 16, 27, 34, int(time.Millisecond*123+time.Microsecond*456+789), time.UTC)
	tm4 := time.Date(1, 1, 1, 14, 25, 32, int(time.Millisecond*123), time.UTC)
	tm5 := time.Date(1, 1, 1, 14, 25, 32, int(time.Millisecond*123), time.UTC)
	tm6 := time.Date(1, 1, 1, 15, 26, 33, int(time.Millisecond*123), time.UTC)
	tm7 := time.Date(1, 1, 1, 16, 27, 34, int(time.Millisecond*123), time.UTC)

	tests := []struct {
		name    string
		r       io.Reader
		want    interface{}
		wantErr bool
	}{
		{
			name: "byte",
			r:    bytes.NewBuffer([]byte{1, 123}),
			want: byte(123),
		},
		{
			name: "short",
			r:    bytes.NewBuffer([]byte{2, 0x39, 0x30}),
			want: int16(12345),
		},
		{
			name: "int",
			r:    bytes.NewBuffer([]byte{3, 0xD2, 0x02, 0x96, 0x49}),
			want: int32(1234567890),
		},
		{
			name: "long",
			r:    bytes.NewBuffer([]byte{4, 0x15, 0x81, 0xE9, 0x7D, 0xF4, 0x10, 0x22, 0x11}),
			want: int64(1234567890123456789),
		},
		{
			name: "float",
			r:    bytes.NewBuffer([]byte{5, 0x65, 0x20, 0xf1, 0x47}),
			want: float32(123456.789),
		},
		{
			name: "double",
			r:    bytes.NewBuffer([]byte{6, 0xad, 0x69, 0x7e, 0x54, 0x34, 0x6f, 0x9d, 0x41}),
			want: float64(123456789.12345),
		},
		{
			name: "char",
			r:    bytes.NewBuffer([]byte{7, 0x41, 0x0}),
			want: Char('A'),
		},
		{
			name: "bool",
			r:    bytes.NewBuffer([]byte{8, 1}),
			want: true,
		},
		{
			name: "string",
			r: bytes.NewBuffer(
				[]byte{9, 0x0B, 0, 0, 0, 0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67}),
			want: "test string",
		},
		{
			name: "UUID",
			r: bytes.NewBuffer([]byte{10, 0x87, 0x46, 0xb1, 0xf8, 0xa7, 0x9d, 0x58, 0xd6, 0xa4,
				0xa4, 0x62, 0x73, 0xdc, 0x2d, 0xbd, 0xb5}),
			want: uid,
		},
		{
			name: "Date",
			r:    bytes.NewBuffer([]byte{11, 0x0, 0xa0, 0xcd, 0x88, 0x62, 0x1, 0x0, 0x0}),
			want: dm,
		},
		{
			name: "byte array",
			r:    bytes.NewBuffer([]byte{12, 3, 0, 0, 0, 1, 2, 3}),
			want: []byte{1, 2, 3},
		},
		{
			name: "short array",
			r:    bytes.NewBuffer([]byte{13, 3, 0, 0, 0, 1, 0, 2, 0, 3, 0}),
			want: []int16{1, 2, 3},
		},
		{
			name: "int array",
			r: bytes.NewBuffer(
				[]byte{14, 3, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}),
			want: []int32{1, 2, 3},
		},
		{
			name: "long array",
			r: bytes.NewBuffer(
				[]byte{15, 3, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0}),
			want: []int64{1, 2, 3},
		},
		{
			name: "float array",
			r: bytes.NewBuffer(
				[]byte{16, 0x3, 0x0, 0x0, 0x0, 0xcd, 0xcc, 0x8c, 0x3f, 0xcd, 0xcc, 0xc, 0x40, 0x33, 0x33, 0x53, 0x40}),
			want: []float32{1.1, 2.2, 3.3},
		},
		{
			name: "double array",
			r: bytes.NewBuffer(
				[]byte{17, 3, 0, 0, 0, 0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0xf1, 0x3f, 0x9a, 0x99,
					0x99, 0x99, 0x99, 0x99, 0x1, 0x40, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0xa, 0x40}),
			want: []float64{1.1, 2.2, 3.3},
		},
		{
			name: "char array",
			r:    bytes.NewBuffer([]byte{18, 3, 0, 0, 0, 0x41, 0x0, 0x42, 0x0, 0x2f, 0x4}),
			want: []Char{'A', 'B', 'Я'},
		},
		{
			name: "bool array",
			r:    bytes.NewBuffer([]byte{19, 3, 0, 0, 0, 1, 0, 1}),
			want: []bool{true, false, true},
		},
		{
			name: "string array",
			r: bytes.NewBuffer([]byte{20, 3, 0, 0, 0,
				0x9, 3, 0, 0, 0, 0x6f, 0x6e, 0x65,
				0x9, 3, 0, 0, 0, 0x74, 0x77, 0x6f,
				0x9, 6, 0, 0, 0, 0xd1, 0x82, 0xd1, 0x80, 0xd0, 0xb8}),
			want: []string{"one", "two", "три"},
		},
		{
			name: "UUID array",
			r: bytes.NewBuffer([]byte{21, 3, 0, 0, 0,
				10, 0xd3, 0x43, 0x2e, 0x7e, 0x4c, 0x7c, 0xc0, 0xa0, 0x81, 0x7c, 0x47, 0x81, 0x68, 0x17, 0xda, 0x8e,
				10, 0xa4, 0x48, 0xf0, 0x72, 0x5f, 0xb5, 0x15, 0x40, 0x27, 0xf6, 0x50, 0x8d, 0x16, 0x64, 0x1, 0x8d,
				10, 0x43, 0x44, 0xd4, 0xc5, 0xf0, 0x1b, 0x7d, 0x82, 0x74, 0xfe, 0x31, 0xde, 0xb5, 0xd8, 0x8, 0x87}),
			want: []uuid.UUID{uid1, uid2, uid3},
		},
		{
			name: "date array",
			r: bytes.NewBuffer([]byte{22, 3, 0, 0, 0,
				11, 0x0, 0xa0, 0xcd, 0x88, 0x62, 0x1, 0x0, 0x0,
				11, 0x0, 0xf0, 0x23, 0x80, 0x6a, 0x1, 0x0, 0x0,
				11, 0x0, 0xf8, 0xc6, 0x81, 0x72, 0x1, 0x0, 0x0}),
			want: []time.Time{dm1, dm2, dm3},
		},
		{
			name: "Timestamp",
			r: bytes.NewBuffer([]byte{33, 0xdb, 0xb, 0xe6, 0x8b, 0x62, 0x1, 0x0, 0x0,
				0x55, 0xf8, 0x6, 0x0}),
			want: tm,
		},
		{
			name: "Timestamp array",
			r: bytes.NewBuffer([]byte{34, 3, 0, 0, 0,
				33, 0xdb, 0xb, 0xe6, 0x8b, 0x62, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0,
				33, 0xa3, 0x38, 0x74, 0x83, 0x6a, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0,
				33, 0x6b, 0x1d, 0x4f, 0x85, 0x72, 0x1, 0x0, 0x0, 0x55, 0xf8, 0x6, 0x0}),
			want: []time.Time{tm1, tm2, tm3},
		},
		{
			name: "Time",
			r:    bytes.NewBuffer([]byte{36, 0xdb, 0x6b, 0x18, 0x3, 0x0, 0x0, 0x0, 0x0}),
			want: tm4,
		},
		{
			name: "Time array",
			r: bytes.NewBuffer([]byte{37, 3, 0, 0, 0,
				36, 0xdb, 0x6b, 0x18, 0x3, 0x0, 0x0, 0x0, 0x0,
				36, 0xa3, 0x48, 0x50, 0x3, 0x0, 0x0, 0x0, 0x0,
				36, 0x6b, 0x25, 0x88, 0x3, 0x0, 0x0, 0x0, 0x0}),
			want: []time.Time{tm5, tm6, tm7},
		},
		{
			name: "NULL",
			r:    bytes.NewBuffer([]byte{101}),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadObject(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("response.ReadObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("response.ReadObject() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
*/
