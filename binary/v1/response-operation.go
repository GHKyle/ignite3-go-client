package ignite3

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"
	"strings"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

const (
	// OperationStatusSuccess means success
	OperationStatusSuccess = 0
	
	FlagPartitionAssignmentChanged = 1
	FlagError                      = 4
)

// ResponseOperation is struct operation response
type ResponseOperation struct {
	// Request id
	RequestId int64
	// Status code (0 for success, otherwise error code)
	Flags     int32
	// Observable Timestamp (causality token)
	ObsTs     time.Time
	// Error message (present only when status is not 0)
	Code      int64
	Uuid      Uuid
	Message   string
	Details   string

	// Copie des données restantes, pour comprendre pourquoi j'ai un EOF :
	// ping failed: failed to execute ping query: failed to read result set column metadata: failed to read result set column property count: EOF
	Data      bytes.Buffer

	response
}

// GRUIIKKK TODO: REMOVE ME
func (r *ResponseOperation) GetMessage() io.Reader {
	return r.message
}

// ReadFrom is function to read request data from io.Reader.
// Returns read bytes.
func (r *ResponseOperation) ReadFrom(rr *bufio.Reader) (int64, error) {
	// read response, put payload in r.message
	n, err := r.response.ReadFrom(rr)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read operation response")
	}

	rid, err := ReadPackedInt64(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read operation response id")
	}

	flags, err := ReadPackedInt64(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read flags")
	}
	r.Flags = int32(flags)
	
	if Debug { fmt.Printf("DEBUG: in ResponseOperation.ReadFrom : flags = %x\n", flags) }

	// When partition assignment changed, a timestamp is added
	if flags & FlagPartitionAssignmentChanged == FlagPartitionAssignmentChanged {
		pcts, err := ReadPackedTimestamp(r.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read partition changed timestamp")
		}
		if Debug { fmt.Printf("DEBUG: Partition changed at %v\n", pcts) }
	}

	// Observable timestamp
	r.ObsTs, err = ReadPackedTimestamp(r.message)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read observable timestamp")
	}

	if int64(rid) != r.RequestId {
		return n, errors.Errorf("invalid request ID: got %d, but expected %d", rid, r.RequestId)
	}

	
	if flags & FlagError == FlagError {
		r.Uuid, err = ReadPackedUUID(r.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read error uuid")
		}

		r.Code, err = ReadPackedInt64(r.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read error code")
		}
		
		r.Message, err = ReadPackedString(r.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read error message")
		}
		
		r.Details, err = ReadPackedString(r.message)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to read error details")
		}

		if Debug {
			fmt.Printf("Erreur at %v, uuid %v :\n", r.ObsTs, r.Uuid)
			fmt.Printf("Code : %x\n", r.Code)
			fmt.Printf("Message : %s\n", r.Message)
			fmt.Printf("Details : %s\n", r.Details)
		}
	}

	return n, nil
}

// CheckStatus checks status of operation execution.
// Returns:
// nil in case of success.
// error object in case of operation failed.
func (r *ResponseOperation) CheckStatus() error {
	// FlagPartitionAssignmentChanged is not an error
	if r.Flags & FlagError == FlagError {
		return errors.NewError(int32(r.Code), strings.Join([]string{r.Message, r.Details}, " : "))
	}
	return nil
}

// NewResponseOperation is ResponseOperation constructor
func NewResponseOperation(uid int64) *ResponseOperation {
	return &ResponseOperation{RequestId: uid}
}
