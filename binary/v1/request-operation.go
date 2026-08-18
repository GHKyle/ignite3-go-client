package ignite3

import (
	"io"
	"bytes"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// Ref : modules/client/src/main/java/org/apache/ignite/internal/client/sql/

// RequestOperation is struct to store operation request
type RequestOperation struct {
	Code      int16
	RequestId int64
	SpecificData []byte

	request
}

// WriteTo is function to write operation request data to io.Writer.
// Returns written bytes.
func (r *RequestOperation) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer

	// First write to buffer, then write buffer length to payload, then write buffer to payload, then write payload on the wire
	// write operation code
	if err := WritePackedInt64(&buf, int64(r.Code)); err != nil {
		return 0, errors.Wrapf(err, "failed to write operation code")
	}
	// write operation request id
	if err := WritePackedInt64(&buf, r.RequestId); err != nil {
		return 0, errors.Wrapf(err, "failed to write request id")
	}
	// Write payload lenth to request
	if err := WriteRawInt32(r, int32(buf.Len() + len(r.SpecificData))); err != nil {
		return 0, errors.Wrapf(err, "failed to write request length")
	}
	
	// Add additionalpayload
	if len(r.SpecificData) > 0 {
		_, err := buf.Write(r.SpecificData)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to write request operation specific data")
		}
	}

	// Write buffer to payload
	buf.WriteTo(r)

	// write payload on the wire
	n, err := r.request.WriteTo(w)

	// Add length length
	return n, err
}

// NewRequestOperation creates new handshake request object
func NewRequestOperation(code int16, requestId int64, payload []byte) *RequestOperation {
	return &RequestOperation{request: newRequest(), Code: code, RequestId: requestId, SpecificData: payload}
}
