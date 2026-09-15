package ignite3

import (
	"bytes"
	"io"
	"sort"
	"strings"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// ---------------------------------------------------------------------------
// Table schema (OP_SCHEMAS_GET)
//
// Request:  int tableId, nil (for the latest schema version) or int count + int version
// Response: int schemaCount, then for each schema:
//   int schemaVersion, int columnCount, then for each column:
//     int propertyCount, string name, int typeId, int keyIndex, bool nullable,
//     int colocationIndex, int scale, int precision [, ...extra properties to skip]
//
// The Java client (ClientTable#readSchema) reads exactly 7 properties and skips the rest,
// which keeps the layout forward compatible.
// ---------------------------------------------------------------------------

// TableColumn describes one column of a table.
type TableColumn struct {
	// Name is the column name as stored in the schema.
	Name string
	// TypeId is the protocol column type id (see the type<X> constants in types.go).
	TypeId int
	// Kind is the matching binary tuple element kind (Bt* constants).
	Kind int
	// KeyIndex is >= 0 for key columns, -1 otherwise. Key columns are ordered by KeyIndex.
	KeyIndex int
	// Key marks key (primary key) columns.
	Key bool
	// Nullable reports whether the column accepts NULL.
	Nullable bool
	// Scale is the decimal scale (DECIMAL only).
	Scale int
	// Precision is the column precision.
	Precision int
}

// TableSchema is a table schema loaded from the cluster.
type TableSchema struct {
	// Version is the schema version to be sent with tuple operations.
	Version int
	// Columns are the columns in schema order.
	Columns []TableColumn
}

// KeyColumns returns the key columns ordered by their key index.
func (s *TableSchema) KeyColumns() []TableColumn {
	var keys []TableColumn
	for _, c := range s.Columns {
		if c.Key {
			keys = append(keys, c)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].KeyIndex < keys[j].KeyIndex })
	return keys
}

// columnKindForTypeId maps a protocol column type id to a binary tuple element kind.
func columnKindForTypeId(typeId int) int {
	switch typeId {
	case typeBOOLEAN:
		return BtBoolean
	case typeTINYINT:
		return BtInt8
	case typeSMALLINT:
		return BtInt16
	case typeINT:
		return BtInt32
	case typeBIGINT:
		return BtInt64
	case typeREAL:
		return BtFloat
	case typeDOUBLE:
		return BtDouble
	case typeDECIMAL:
		return BtDecimal
	case typeDATE:
		return BtDate
	case typeTIME:
		return BtTime
	case typeDATETIME:
		return BtDateTime
	case typeTIMESTAMP:
		return BtTimestamp
	case typeUUID:
		return BtUuid
	case typeVARCHAR:
		return BtString
	case typeVARBINARY:
		return BtBytes
	default:
		return BtNull
	}
}

// GetTableSchema loads the latest schema of a table.
func (c *client) GetTableSchema(tableId int) (*TableSchema, error) {
	var buf bytes.Buffer
	if err := WritePackedInt64(&buf, int64(tableId)); err != nil {
		return nil, errors.Wrapf(err, "failed to write table id")
	}
	// nil means "give me the latest schema version".
	if err := WritePackedNull(&buf); err != nil {
		return nil, errors.Wrapf(err, "failed to write schema version marker")
	}

	req := NewRequestOperation(OpSchemaGet, c.RequestId, buf.Bytes())
	res := NewResponseOperation(req.RequestId)

	if err := c.Do(req, res); err != nil {
		return nil, errors.Wrapf(err, "failed to execute GET_SCHEMA operation")
	}
	if err := res.CheckStatus(); err != nil {
		return nil, err
	}

	count, err := ReadPackedInt32(res.message)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read schema count")
	}
	if count <= 0 {
		return nil, errors.Errorf("schema not found for table id %d", tableId)
	}

	var last *TableSchema
	for i := 0; i < int(count); i++ {
		version, err := ReadPackedInt32(res.message)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read schema version")
		}

		columnCount, err := ReadPackedInt32(res.message)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read column count")
		}

		columns := make([]TableColumn, 0, columnCount)
		for j := 0; j < int(columnCount); j++ {
			propertyCount, err := ReadPackedInt32(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column property count")
			}

			name, err := ReadPackedString(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column name")
			}
			typeId, err := ReadPackedInt32(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column type")
			}
			keyIndex, err := ReadPackedInt32(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column key index")
			}
			nullable, err := ReadPackedBool(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column nullability")
			}
			if _, err := ReadPackedInt32(res.message); err != nil { // colocation index
				return nil, errors.Wrapf(err, "failed to read column colocation index")
			}
			scale, err := ReadPackedInt32(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column scale")
			}
			precision, err := ReadPackedInt32(res.message)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to read column precision")
			}

			// Skip properties added by newer servers.
			for k := 7; k < int(propertyCount); k++ {
				if err := skipPackedValue(res.message); err != nil {
					return nil, errors.Wrapf(err, "failed to skip column property")
				}
			}

			columns = append(columns, TableColumn{
				Name:      name,
				TypeId:    int(typeId),
				Kind:      columnKindForTypeId(int(typeId)),
				KeyIndex:  int(keyIndex),
				Key:       keyIndex >= 0,
				Nullable:  nullable,
				Scale:     int(scale),
				Precision: int(precision),
			})
		}

		last = &TableSchema{Version: int(version), Columns: columns}
	}

	return last, nil
}

// GetTableByName resolves a table by its qualified name ("SCHEMA"."TABLE" or "SCHEMA.TABLE",
// quoting optional, case insensitive) and loads its schema. The cluster may report either the
// qualified name or the bare table name, so an exact match is preferred and a suffix match is
// used as a fallback.
func (c *client) GetTableByName(qualifiedName string) (*Table, error) {
	wanted := normalizeTableName(qualifiedName)

	tables, err := c.GetTables()
	if err != nil {
		return nil, err
	}

	var candidates []int64
	for id, name := range tables {
		reported := normalizeTableName(name)
		if reported == wanted {
			candidates = append([]int64{id}, candidates...)
		} else if strings.HasSuffix(wanted, "."+reported) || strings.HasSuffix(reported, "."+wanted) {
			candidates = append(candidates, id)
		}
	}

	if len(candidates) == 0 {
		names := make([]string, 0, len(tables))
		for _, name := range tables {
			names = append(names, name)
		}
		sort.Strings(names)
		return nil, errors.Errorf("table %s not found, known tables: %v", qualifiedName, names)
	}

	id := candidates[0]
	schema, err := c.GetTableSchema(int(id))
	if err != nil {
		return nil, err
	}

	return &Table{Id: int(id), Name: tables[id], Schema: schema, c: c}, nil
}

// normalizeTableName lowercases a table name and strips quoting and whitespace so that
// "SCHEMA"."TABLE", schema.table and table can all be compared.
func normalizeTableName(name string) string {
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "`", "")
	parts := strings.Split(name, ".")
	for i := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
	}
	return strings.Join(parts, ".")
}

// skipPackedValue skips one msgpack value, used for forward compatibility.
func skipPackedValue(r io.Reader) error {
	return msgpack.NewDecoder(r).Skip()
}
