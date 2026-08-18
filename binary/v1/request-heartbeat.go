package ignite3

import (
	"io"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// RequestHandshake is struct handshake request
type RequestHeartbeat struct {
	Code int8
	UID  int64

	request
}

// WriteTo is function to write handshake request data to io.Writer.
// Returns written bytes.
func (r *RequestHeartbeat) WriteTo(w io.Writer) (int64, error) {
	if err := WriteRawInt32(r, int32(2)); err != nil {
		return 0, errors.Wrapf(err, "failed to write heartbeat request length")
	}
	if err := WritePackedInt64(r, int64(r.Code)); err != nil {
		return 0, errors.Wrapf(err, "failed to write heartbeat operation code")
	}
	if err := WritePackedInt64(r, r.UID); err != nil {
		return 0, errors.Wrapf(err, "failed to write heartbeat id")
	}

	// write request
	n, err := r.request.WriteTo(w)
	return n, err
}

// NewRequestHeartbeat creates new heartbeat request object
func NewRequestHeartbeat(id int64) *RequestHeartbeat {
	r := RequestHeartbeat{request: newRequest(),
		Code: OpHeartbeat,
		UID: id}
	return &r
}
