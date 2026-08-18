package ignite3

import (
	"bufio"
	"bytes"
	//"io"
	"testing"
)

func TestResponseHandshake_ReadFrom(t *testing.T) {
	rr1 := bytes.NewBuffer(
		[]byte{0x49, 0x47, 0x4e, 0x49, 0x00, 0x00, 0x00, 0x4a, 0x03, 0x00, 0x00, 0xc0, 0x00, 0xd8, 0x03, 0xed, 
			0x46, 0x19, 0xe9, 0x07, 0x72, 0x55, 0x84, 0x4b, 0x71, 0x1d, 0x09, 0x16, 0x94, 0x22, 0x87, 0xa5, 
			0x6e, 0x6f, 0x64, 0x65, 0x31, 0x01, 0xd8, 0x03, 0x8c, 0x4f, 0xb7, 0x26, 0x62, 0x74, 0x91, 0x1e,
			0x02, 0xee, 0x3e, 0x9d, 0xf3, 0xd5, 0xeb, 0x9d, 0xa6, 0x74, 0x65, 0x73, 0x74, 0x30, 0x31, 0xcf,
			0x01, 0x9b, 0xa8, 0x32, 0x1f, 0xbc, 0x00, 0x00, 0x03, 0x01, 0x00, 0xc0, 0xc0, 0xc4, 0x02, 0x2a, 
			0x0a, 0x00})
	r1 := &ResponseHandshake{}

	type args struct {
		rr *bufio.Reader
	}
	tests := []struct {
		name                            string
		r                               *ResponseHandshake
		args                            args
		want                            int64
		wantSuccess                     bool
		wantMajor, wantMinor, wantPatch int
		wantMessage                     string
		wantErr                         bool
	}{
		{
			name: "1",
			r:    r1,
			args: args{
				rr: bufio.NewReader(rr1),
			},
			want:        3 + 1 + 1 + 18 + 1 + 5 + 1 + 18 + 1 + 6 + 1 + 8 + 1 + 1 + 1 + 1 + 1 + 1 + 4,
			wantSuccess: true,
			wantMajor:   3,
			wantMinor:   0,
			wantPatch:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.ReadFrom(tt.args.rr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResponseHandshake.ReadFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResponseHandshake.ReadFrom() = %v", got)
				t.Errorf("ResponseHandshake.ReadFrom() want = %v", tt.want)
			}
			if tt.r.Success != tt.wantSuccess {
				t.Errorf("ResponseHandshake.ReadFrom() success = %v, want %v", tt.r.Success, tt.wantSuccess)
			}
			if tt.r.Major != tt.wantMajor {
				t.Errorf("ResponseHandshake.ReadFrom() major = %v, want %v", tt.r.Major, tt.wantMajor)
			}
			if tt.r.Minor != tt.wantMinor {
				t.Errorf("ResponseHandshake.ReadFrom() minor = %v, want %v", tt.r.Minor, tt.wantMinor)
			}
			if tt.r.Patch != tt.wantPatch {
				t.Errorf("ResponseHandshake.ReadFrom() patch = %v, want %v", tt.r.Patch, tt.wantPatch)
			}
			if tt.r.ErrorMessage != tt.wantMessage {
				t.Errorf("ResponseHandshake.ReadFrom() message = %v, want %v", tt.r.ErrorMessage, tt.wantMessage)
			}
		})
	}
}
