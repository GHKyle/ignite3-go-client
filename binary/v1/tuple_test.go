package ignite3

import (
	"reflect"
	"testing"
)

func TestNewTupleGetValue(t *testing.T) {
	type args struct {
		meta  ResultSetMetadata
		bufr  []byte
		index int
		valueType int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				meta: ResultSetMetadata{Columns: []ColumnMetadata{
					ColumnMetadata{
						Name: "TRACKID", 
						Nullable:false,
						Type: &ColumnType{
						},
						Precision: 10,
						Scale:      0,
						Index:      0,
						Origin: &ColumnOrigin{
							Schema: "PUBLIC",
							Table:  "TRACK", 
							Column: "TRACKID",
						},
					},
					ColumnMetadata{
						Name: "NAME", 
						Nullable:false,
						Type: &ColumnType{
						},
						Precision: 200,
						Scale:       0,
						Index:       1,
						Origin: &ColumnOrigin{
							Schema: "PUBLIC",
							Table:  "TRACK", 
							Column: "NAME",
						},
					},
				}},
				bufr: []byte{0x00,0x02,0x11,0xd0,0x04,0x48,0x65,0x61,0x76,0x65,0x6e,0x20,0x43,0x61,0x6e,0x20,0x57,0x61,0x69,0x74},
				index: 1,
				valueType: typeVARCHAR,
			},
			want: "Heaven Can Wait",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tuple *Tuple
			var err error
			if tuple, err = NewTupleFromByteArrayWithMetadata(&tt.args.meta, tt.args.bufr); (err != nil) != tt.wantErr {
				t.Errorf("TestNewTupleGetValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			var value interface{}
			if value, err = tuple.GetValue(tt.args.index, tt.args.valueType); err != nil {
				t.Errorf("TestNewTupleGetValue() error in GetValue = %v", err)
			}
			if !reflect.DeepEqual(value, tt.want) {
				t.Errorf("TestNewTupleGetValue() = %#v, want %#v", value, tt.want)
			}
		})
	}
}
