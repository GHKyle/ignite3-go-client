package ignite3

import (
	"bytes"
	"strings"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// ---------------------------------------------------------------------------
// Tuple (key-value) operations: OP_TUPLE_UPSERT_ALL / OP_TUPLE_DELETE_ALL
//
// These operations send rows as binary tuples instead of SQL statements, so the server
// performs a plain key-based write: no SQL parsing, no planning, no plan cache entries and
// no table scan. For CDC / bulk loading this is dramatically cheaper than MERGE, whose plan
// is a hash join over a full table scan (verified with EXPLAIN on Ignite 3.1).
//
// Request payload (see the Java client ClientTupleSerializer#writeTuples):
//
//	int  tableId
//	nil                     // no transaction: DirectTxUtils#writeTx writes nil for a null tx
//	int  schemaVersion
//	int  rowCount
//	rowCount times:
//	    nil                 // "no value" bit set; nil when every column is present
//	    binary(tupleBytes)  // see BuildBinaryTuple
// ---------------------------------------------------------------------------

// bitmaskExtType is ClientMsgPackType.BITMASK from the Ignite 3 client protocol.
const bitmaskExtType = 8

// writePackedEmptyBitSet writes an empty bit set as a msgpack extension (ext8, zero length).
func writePackedEmptyBitSet(buf *bytes.Buffer) error {
	buf.WriteByte(0xc7) // msgpack ext8
	buf.WriteByte(0x00) // extension payload length
	buf.WriteByte(bitmaskExtType)
	return nil
}

// Table is a handle to an Ignite table with its schema loaded, used for binary tuple operations.
type Table struct {
	// Id is the table id.
	Id int
	// Name is the table name as reported by the cluster.
	Name string
	// Schema is the current schema of the table.
	Schema *TableSchema

	c *client
}

// SchemaVersion returns the schema version to be sent with tuple operations.
func (t *Table) SchemaVersion() int { return t.Schema.Version }

// TableUpsertAll inserts or overwrites rows by key. Every row must be a full binary tuple
// produced by BuildBinaryTuple using the table columns in schema order.
func (c *client) TableUpsertAll(tableId int, schemaVersion int, rows [][]byte) error {
	return c.tupleWriteAll(OpTupleUpsertAll, tableId, schemaVersion, rows)
}

// TableDeleteAll deletes rows by key. Every key must be a binary tuple built from the table
// key columns in key order.
func (c *client) TableDeleteAll(tableId int, schemaVersion int, keys [][]byte) error {
	return c.tupleWriteAll(OpTupleDeleteAll, tableId, schemaVersion, keys)
}

func (c *client) tupleWriteAll(opCode int16, tableId int, schemaVersion int, tuples [][]byte) error {
	if len(tuples) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := WritePackedInt64(&buf, int64(tableId)); err != nil {
		return errors.Wrapf(err, "failed to write table id")
	}
	// No transaction.
	if err := WritePackedNull(&buf); err != nil {
		return errors.Wrapf(err, "failed to write transaction marker")
	}
	if err := WritePackedInt64(&buf, int64(schemaVersion)); err != nil {
		return errors.Wrapf(err, "failed to write schema version")
	}
	if err := WritePackedInt64(&buf, int64(len(tuples))); err != nil {
		return errors.Wrapf(err, "failed to write row count")
	}
	for i, tuple := range tuples {
		// The "no value" set marks columns that are absent from the tuple (as opposed to columns
		// explicitly set to NULL). This API always provides every column, so the set is empty.
		// It must still be written: the server reads it as a msgpack extension of type
		// ClientMsgPackType.BITMASK (8), and writing nil instead fails with "Expected Ext, but got Nil".
		if err := writePackedEmptyBitSet(&buf); err != nil {
			return errors.Wrapf(err, "failed to write value mask of row %d", i)
		}
		if err := WritePackedBytes(&buf, tuple); err != nil {
			return errors.Wrapf(err, "failed to write tuple of row %d", i)
		}
	}

	req := NewRequestOperation(opCode, c.RequestId, buf.Bytes())
	res := NewResponseOperation(req.RequestId)

	if err := c.Do(req, res); err != nil {
		return errors.Wrapf(err, "failed to execute tuple operation %d", opCode)
	}

	return res.CheckStatus()
}

// UpsertAll inserts or overwrites the given rows (full rows, identified by the primary key).
// Each row maps a column name (case insensitive) to a value; missing columns are written as NULL.
func (t *Table) UpsertAll(rows []map[string]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	tuples := make([][]byte, 0, len(rows))
	for i, row := range rows {
		tuple, err := t.buildTuple(row, false)
		if err != nil {
			return errors.Wrapf(err, "failed to encode row %d", i)
		}
		tuples = append(tuples, tuple)
	}

	return t.c.TableUpsertAll(t.Id, t.Schema.Version, tuples)
}

// DeleteAll deletes the rows identified by the given keys.
func (t *Table) DeleteAll(keys []map[string]interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	tuples := make([][]byte, 0, len(keys))
	for i, key := range keys {
		tuple, err := t.buildTuple(key, true)
		if err != nil {
			return errors.Wrapf(err, "failed to encode key %d", i)
		}
		tuples = append(tuples, tuple)
	}

	return t.c.TableDeleteAll(t.Id, t.Schema.Version, tuples)
}

// buildTuple encodes one row (keyOnly == false) or one key (keyOnly == true) as a binary tuple.
func (t *Table) buildTuple(row map[string]interface{}, keyOnly bool) ([]byte, error) {
	columns := t.Schema.Columns
	if keyOnly {
		columns = t.Schema.KeyColumns()
	}

	lookup := make(map[string]interface{}, len(row))
	for k, v := range row {
		lookup[strings.ToUpper(k)] = v
	}

	cols := make([]BinaryTupleColumn, len(columns))
	vals := make([]interface{}, len(columns))
	for i, col := range columns {
		cols[i] = BinaryTupleColumn{Kind: col.Kind, Scale: col.Scale}
		vals[i] = lookup[strings.ToUpper(col.Name)]
	}

	return BuildBinaryTuple(cols, vals)
}
