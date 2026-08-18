package common

import (
	"github.com/GHKyle/ignite3-go-client/binary/v1"
)

// ConnInfo contains Apache Ignite cluster connection and query execution parameters
type ConnInfo struct {
	URL string

	// Apache Ignite client connection information
	ignite3.ConnInfo

	Cache string

	// Schema for the query; can be empty, in which case default PUBLIC schema will be used.
	Schema string

	// Query cursor page size.
	PageSize int

	// Timeout(milliseconds) value should be non-negative. Zero value disables timeout.
	Timeout int64
}
