package v1

import (
	"fmt"
	"context"
	"database/sql/driver"
	"io"
	"reflect"
	"runtime"
	"time"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
	ignite3 "github.com/GHKyle/ignite3-go-client/binary/v1"
	"github.com/GHKyle/ignite3-go-client/debug"
)

type columnType struct {
	name    string
	sqlType string
}

// Rows is an iterator over an executed query's results.
type Rows interface {
	driver.Rows
}

type rows struct {
	conn     *conn
	response *ignite3.ResponseOperation
	metadata *ignite3.ResultSetMetadata
	id       int64
	fields   []columnType
	rowsLeft int
	hasMore  bool
}

// Columns returns the names of the columns. The number of
// columns of the result is inferred from the length of the
// slice. If a particular column name isn't known, an empty
// string should be returned for that entry.
func (r *rows) Columns() []string {
	columns := make([]string, len(r.fields))
	for i, f := range r.fields {
		columns[i] = f.name
	}
	return columns
}

func (r *rows) ColumnTypeDatabaseTypeName(i int) string {
	return r.fields[i].sqlType
}

// Close closes the rows iterator.
func (r *rows) Close() error {
	if r.rowsLeft > 0 {
		// to prevent resource leak on server try to close cursor
		r.rowsLeft = 0
		return r.conn.resourceClose(r.id)
	}
	return nil
}

// Next is called to populate the next row of data into
// the provided slice. The provided slice will be the same
// size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
//
// The dest should not be written to outside of Next. Care
// should be taken when closing Rows not to modify
// a buffer held in dest.
func (r *rows) Next(dest []driver.Value) error {
	var err error
	if r.rowsLeft == 0 {
		if !r.hasMore {
			return io.EOF
		}
		if r.response, err = r.conn.QueryNexPageContext(context.Background(), r.id); err != nil {
			// prevent resource leak on server
			_ = r.Close()
			return errors.Wrapf(err, "failed to read cursor page")
		}

		// read data
		var rowCount int64
		if rowCount, err = ignite3.ReadPackedInt(r.response.GetMessage()); err != nil {
			// prevent resource leak on server
			_ = r.Close()
			return errors.Wrapf(err, "failed to read row count")
		}
		r.rowsLeft = int(rowCount)
	}
	if len(r.fields) != len(dest) {
		return errors.Errorf("destination slice size must be %d but got %d", len(r.fields), len(dest))
	}

	bytes, err := ignite3.ReadPackedBytes(r.response.GetMessage())
	if err != nil {
		return errors.Wrapf(err, "failed to read row tuple")
	}

	var tuple *ignite3.Tuple
	if tuple, err = ignite3.NewTupleFromByteArrayWithMetadata(r.metadata, bytes); err != nil {
		return errors.Wrapf(err, "failed to parse row tuple")
	}

	var ctype *ignite3.ColumnType
	var value interface{}
	for i := 0; i < len(r.metadata.Columns); i++ {
		if ctype, err = r.metadata.GetColumnTypeById(i); err != nil {
			return errors.Wrapf(err, "failed to get column type parsing rows")
		}
		if value, err = tuple.GetValue(i, ctype.TypeId); err != nil {
			fmt.Printf("Erreur dans GetValue : %v\n", err)
			return errors.Wrapf(err, "failed to parse column %d", i)
		}
		// Convert value if needed
		switch ctype.GoKind {
			case reflect.Struct:
				switch ctype.SQLTypeName {
					case "DECIMAL":
						value = value.(ignite3.Decimal).String()
					case "UUID":
						value = value.(ignite3.Uuid).String()
					case "DATETIME", "TIMESTAMP":
						value = value.(time.Time).String()
					default:
						return errors.Errorf("Unsupported column type: %s at %d", ctype.SQLTypeName, i)
				}
		}
		dest[i] = value
	}
	r.rowsLeft--

	return nil
}

// newRows creates new Rows object
func newRows(conn *conn, r *ignite3.ResponseOperation) (driver.Rows, error) {
	var err error
	var result ignite3.QuerySQLResult

	// Read resource Id, hasRowSet, hasMore, wasApplied, affectedRows
	if result, err = ignite3.ReadQuerySQLResponseHeader(result, r); err != nil {
		return nil, err
	}

	// Read ResultSet MetaData which contains schema
	if result.Metadata, err = ignite3.NewResultSetMetadataFromReader(r.GetMessage()); err != nil {
		return nil, err
	}

	// Build fields from column names
	fields := make([]columnType, 0, len(result.Metadata.Columns))
	for _, c := range result.Metadata.Columns {
		fields = append(fields, columnType{name: c.Name, sqlType: c.Type.SQLTypeName})
	}

	// Skip partition awareness (Should we suuport it ? Is this necessary ?)
	_, _ = ignite3.ReadPartitionAwareness(r, false, false)
	
	var rowCount int64
	if (result.HasRowSet) {
		if rowCount, err = ignite3.ReadPackedInt64(r.GetMessage()); err != nil {
			return nil, errors.Wrapf(err, "failed to read row count")
		}
	} else {
		rowCount = 0
	}

	rs := &rows{
		conn:     conn,
		response: r,
		metadata: result.Metadata,
		id:       result.ResponseId,
		fields:   fields,
		rowsLeft: int(rowCount),
		hasMore:  result.HasMore,
	}
	runtime.SetFinalizer(rs, rowsFinalizer)
	return rs, nil
}

// connFinalizer is memory leak spy
func rowsFinalizer(r *rows) {
	if r.rowsLeft > 0 {
		debug.ResourceLeakLogger.Printf("rows with cursor ID=\"%d\" is not closed", r.id)
		r.Close()
	}
}
