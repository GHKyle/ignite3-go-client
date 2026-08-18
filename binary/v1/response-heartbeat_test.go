package ignite3

import (
	"bufio"
	"bytes"
	//"io"
	"testing"
)

// v3_OK
func TestResponseHeartbeat_ReadFrom(t *testing.T) {
	// v3_TODO : Real data here
	rr1 := bufio.NewReader(bytes.NewBuffer(
		[]byte{0, 0, 0, 0xb, 0x0, 0x0, 0xd3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}))
	rr2 := bufio.NewReader(bytes.NewBuffer(
		[]byte{0, 0, 0, 0xb, 0x4, 0x0, 0xd3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}))

	r1 := &ResponseHeartbeat{}
	r2 := &ResponseHeartbeat{}

	type args struct {
		rr *bufio.Reader
	}
	tests := []struct {
		name                            string
		r                               *ResponseHeartbeat
		args                            args
		want                            int64
		wantStatus                      int32
		wantUID                         int64
		wantErr                         bool
	}{
		{
			name: "1",
			r:    r1,
			args: args{
				rr: rr1,
			},
			want:        4 + 11,
			wantStatus:  0,
			wantUID:     0,
			wantErr:     false,
		},
		{
			name: "2",
			r:    r2,
			args: args{
				rr: rr2,
			},
			want:        4 + 11,
			wantStatus:  0,
			wantUID:     0,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.ReadFrom(tt.args.rr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResponseHeartbeat.ReadFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResponseHeartbeat.ReadFrom() = %v, want %v", got, tt.want)
			}
			if tt.r.UID != tt.wantUID {
				t.Errorf("ResponseHeartbeat.ReadFrom() UID = %v, want %v", tt.r.UID, tt.wantUID)
			}
			if tt.r.Status != tt.wantStatus {
				t.Errorf("ResponseHeartbeat.ReadFrom() status = %v, want %v", tt.r.Status, tt.wantStatus)
			}
		})
	}
}
