package ignite3

// v3_OK (all file)
import (
	"bufio"
	//"fmt"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

const (
	// HeartbeatStatusSuccess means success
	HeartbeatStatusSuccess = 0
)

// ResponseOperation is struct operation response
type ResponseHeartbeat struct {
	// Request id
	UID int64
	// Status code (0 for success, otherwise error code)
	Status int32
	// What is this data?
	UnknownData int64

	response
}

// ReadFrom is function to read request data from io.Reader.
// Returns read bytes.
// heartbeat request  : 00000002 01 04
// heartbeat response : 0000000b 04 00 d3 00 00 00 00 00 00 00
func (r *ResponseHeartbeat) ReadFrom(rr *bufio.Reader) (int64, error) {
	// read response, put payload in r.message
	n, err := r.response.ReadFrom(rr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read operation response")
	}
	//fmt.Printf("DEBUG: payload length: %d\n", n)

	uid, err := ReadPackedInt8(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read operation response id")
	}

	status, err := ReadPackedInt8(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read status code")
	}
	r.Status = int32(status)
	//fmt.Printf("DEBUG: status = %v of type %T\n", status, status)
	
	r.UnknownData, err = ReadPackedInt64(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read unkown data")
	}

	if r.Status != HeartbeatStatusSuccess {
		return 0, errors.Errorf("failed to read error message")
	}

	if int64(uid) != r.UID {
		return n, errors.Errorf("invalid request ID: got %d, but expected %d", uid, r.UID)
	}

	return n, nil
}

// CheckStatus checks status of operation execution.
// Returns:
// nil in case of success.
// error object in case of operation failed.
func (r *ResponseHeartbeat) CheckStatus() error {
	if r.Status != HeartbeatStatusSuccess {
		return errors.NewError(r.Status, "")
	}
	return nil
}

// NewResponseOperation is ResponseHeartbeat constructor
func NewResponseHeartbeat(uid int64) *ResponseHeartbeat {
	return &ResponseHeartbeat{UID: uid}
}
