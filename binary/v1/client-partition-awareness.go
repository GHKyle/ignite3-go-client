package ignite3

import (
	"io"
	"fmt"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

type ClientDirectTxMode uint8

const (
	ClientDirectTxModeNotSupported ClientDirectTxMode = iota
	ClientDirectTxModeSupported
	ClientDirectTxModeSupportedTrackingRequired
)

func NewClientDirectTxModeFromByte(val uint8) ClientDirectTxMode {
	if val < 0 || ClientDirectTxMode(val) > ClientDirectTxModeSupportedTrackingRequired {
		return ClientDirectTxModeNotSupported
	} else {
		return ClientDirectTxMode(val)
	}
}

type ClientPartitionAwarenessMetadata struct {
	TableId      int
	Indexes      []int
	Hash         []int
	DirectTxMode ClientDirectTxMode
}

func NewClientPartitionAwarenessMetadataFromReader(r io.Reader, sqlDirectMappingSupported bool) (*ClientPartitionAwarenessMetadata, error) {
	var cpam ClientPartitionAwarenessMetadata

	ti, err := ReadPackedInt64(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read result set partition awareness table ID")
	}
	cpam.TableId = int(ti)
	
	idx, err := ReadPackedInt64Array(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read result set partition awareness indices")
	}
	for _, i := range idx {
		cpam.Indexes = append(cpam.Indexes, int(i))
	}
	
	h, err := ReadPackedInt64Array(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read result set partition awareness hash")
	}
	for _, i := range h {
		cpam.Hash = append(cpam.Hash, int(i))
	}
	
	cpam.DirectTxMode = ClientDirectTxModeNotSupported
	if sqlDirectMappingSupported {
		ctxm, err := ReadPackedByte(r)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set partition awareness direct TX mode support")
		}
		cpam.DirectTxMode = NewClientDirectTxModeFromByte(uint8(ctxm))
	}
	
	fmt.Printf("DEBUG: in NewClientPartitionAwarenessMetadataFromReader: cpam = %v\n", cpam)
	
	return &cpam, nil
}
