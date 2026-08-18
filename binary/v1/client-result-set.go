package ignite3

// Test des types avec :
/*
CREATE TABLE TEST_TYPES(BOOL BOOLEAN, TINT TINYINT PRIMARY KEY, SINT SMALLINT, INT INT, BINT BIGINT, REAL REAL, DOUBLE DOUBLE, TSTAMP TIMESTAMP, LTSTAMP TIMESTAMP WITH LOCAL TIME ZONE, DAT DATE, TIM TIME, UID UUID, VARCHAR VARCHAR, VARBINARY VARBINARY);
INSERT INTO TEST_TYPES VALUES (true, 0, 1, 2, 3, 4.3, 5, LOCALTIMESTAMP, CURRENT_TIMESTAMP, CURRENT_DATE, LOCALTIME, RAND_UUID(), 'six', x'801234');
INSERT INTO TEST_TYPES VALUES (true, 1, 7, 8, 9, 10.1, 11, LOCALTIMESTAMP, CURRENT_TIMESTAMP, CURRENT_DATE, LOCALTIME, RAND_UUID(), '', x'6789abcd');
INSERT INTO TEST_TYPES VALUES (false, 2, 12, 13, 14, 15.1, 16754353475675.43, LOCALTIMESTAMP, CURRENT_TIMESTAMP, CURRENT_DATE, LOCALTIME, RAND_UUID(), 'test', x'');

*/

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"encoding/json"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

// ColumnMetadata.java & ColumnType.java

const (
	UNDEFINED_PRECISION = -1
	UNDEFINED_SCALE     = math.MinInt
)

/**
 * Represents a column origin.
 *
 * <p>Example:
 * <pre>
 *     SELECT SUM(price), category as cat, subcategory AS subcategory
 *       FROM Goods
 *      WHERE [condition]
 *      GROUP BY cat, subcategory
 * </pre>
 *
 * <p>Column origins:
 * <ul>
 * <li>SUM(price): null</li>
 * <li>cat: {"PUBLIC", "Goods", "category"}</li>
 * <li>subcategory: {"PUBLIC", "Goods", "subcategory"}</li>
 * </ul>
 */
type ColumnOrigin struct {
	Schema string
	Table  string
	Column string
}

type ColumnMetadata struct {
	Name      string
	Nullable  bool
	//Class<?> valueClass();
	Type      *ColumnType  // See columnType.java
	Precision int
	Scale     int
	Index     int
	Origin    *ColumnOrigin
}

type ResultSetMetadata struct {
	Columns []ColumnMetadata
}

type ColumnType struct {
	SQLTypeName      string        // Type name in SQL
	TypeId           int           // Type ID in Ignite3
	GoKind           reflect.Kind  // Kind name in Golang
	PrecisionAllowed bool
	ScaleAllowed     bool
	LengthAllowed    bool
}

type ColumnTypes []ColumnType

var (
	ColTypes = ColumnTypes {
		ColumnType{
			SQLTypeName:      "NULL",
			TypeId:           typeNULL,
			GoKind:           reflect.Pointer, // ?          
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "BOOLEAN",
			TypeId:           1,
			GoKind:           typeBOOLEAN,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "TINYINT",
			TypeId:           typeTINYINT,
			GoKind:           reflect.Int8,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "SMALLINT",
			TypeId:           typeSMALLINT,
			GoKind:           reflect.Int16,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "INT",
			TypeId:           typeINT,
			GoKind:           reflect.Int32,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "BIGINT",
			TypeId:           typeBIGINT,
			GoKind:           reflect.Int64,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "REAL",
			TypeId:           typeREAL,
			GoKind:           reflect.Float32,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		ColumnType{
			SQLTypeName:      "DOUBLE",
			TypeId:           typeDOUBLE,
			GoKind:           reflect.Float64,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/** Arbitrary-precision signed decimal number. SQL type: {@code DECIMAL}. */
		ColumnType{
			SQLTypeName:      "DECIMAL",
			TypeId:           typeDECIMAL,
			GoKind:           reflect.Struct,      // ??
			PrecisionAllowed: true,
			ScaleAllowed:     true,
			LengthAllowed:    false,
		},
		/** Timezone-free date. SQL type: {@code DATE}. */
		ColumnType{
			SQLTypeName:      "DATE",
			TypeId:           typeDATE,
			GoKind:           reflect.Int,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/** Timezone-free time with precision. SQL type: {@code TIME}. */
		ColumnType{
			SQLTypeName:      "TIME",
			TypeId:           typeTIME,
			GoKind:           reflect.Int,
			PrecisionAllowed: true,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/** Timezone-free datetime. SQL type: {@code TIMESTAMP}. */
		ColumnType{
			SQLTypeName:      "DATETIME",
			TypeId:           typeDATETIME,
			GoKind:           reflect.Struct,
			PrecisionAllowed: true,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/** Point on the time-line. Number of ticks since {@code 1970-01-01T00:00:00Z}. Tick unit depends on precision. SQL type: {@code TIMESTAMP WITH LOCAL TIME ZONE}. */
		ColumnType{
			SQLTypeName:      "TIMESTAMP",
			TypeId:           typeTIMESTAMP,
			GoKind:           reflect.Struct,
			PrecisionAllowed: true,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/** 128-bit UUID.  SQL type: {@code UUID}. */
		ColumnType{
			SQLTypeName:      "UUID",
			TypeId:           typeUUID,
			GoKind:           reflect.Struct,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},
		/* Unused
		ColumnType{
			SQLTypeName:      "UNUSED",
			TypeId:           14,
			GoKind:           reflect.UNUSED,
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    false,
		},*/
		ColumnType{
			SQLTypeName:      "VARCHAR",
			TypeId:           typeVARCHAR,
			GoKind:           reflect.String,
			PrecisionAllowed: false,            // VARCHAR length is stored in Precision in result tuple
			ScaleAllowed:     false,
			LengthAllowed:    true,
		},
		ColumnType{
			SQLTypeName:      "VARBINARY",
			TypeId:           typeVARBINARY,
			GoKind:           reflect.Array,      // ??
			PrecisionAllowed: false,
			ScaleAllowed:     false,
			LengthAllowed:    true,
		},
	}
)

func (c ColumnTypes) GetColumnTypeById(id int) (*ColumnType, error) {
	for _, t := range c {
		if t.TypeId == id {
			return &t, nil
		}
	}
	return nil, errors.Errorf("type not found")
}

func (c ColumnTypes) GetKindByName(name string) (reflect.Kind, error) {
	for _, t := range c {
		if strings.EqualFold(t.SQLTypeName, name) {
			return t.GoKind, nil
		}
	}
	return reflect.Invalid, errors.Errorf("type not found")
}

func (c ColumnTypes) GetTypeIdBySQLName(name string) (int, error) {
	for _, t := range c {
		if strings.EqualFold(t.SQLTypeName, name) {
			return t.TypeId, nil
		}
	}
	return 0, errors.Errorf("type not found")
}

func (c ColumnTypes) GetTypeIdByGoKind(name string) (int, error) {
	for _, t := range c {
		if strings.EqualFold(t.SQLTypeName, name) {
			return t.TypeId, nil
		}
	}
	return 0, errors.Errorf("type not found")
}

func (c ColumnTypes) GetKindById(id int) (reflect.Kind, error) {
	for _, t := range c {
		if t.TypeId == id {
			return t.GoKind, nil
		}
	}
	return reflect.Invalid, errors.Errorf("type not found")
}

func (c ColumnTypes) GetNameById(id int) (string, error) {
	for _, t := range c {
		if t.TypeId == id {
			return t.SQLTypeName, nil
		}
	}
	return "", errors.Errorf("type not found")
}

func (c *ColumnOrigin) ToJsonString() string {
	js, _ := json.Marshal(c)
	return string(js)
}

func (c *ColumnMetadata) ToJsonString() string {
	js, _ := json.Marshal(c)
	return string(js)
}


func (r *ResultSetMetadata) IndexOf(column string) int {
	for _, c := range r.Columns {
		if strings.EqualFold(c.Name, column) {
			return c.Index
		}
	}
	return -1
}

func (r *ResultSetMetadata) GetColumnOriginNameById(index int) (string, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Origin.Column, nil
		}
	}
	return "", errors.Errorf("failed to find column origin name from index")
}

func (r *ResultSetMetadata) GetSchemaOriginNameById(index int) (string, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Origin.Schema, nil
		}
	}
	return "", errors.Errorf("failed to find column origin schema from index")
}

func (r *ResultSetMetadata) GetTableOriginNameById(index int) (string, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Origin.Table, nil
		}
	}
	return "", errors.Errorf("failed to find column origin table from index")
}

func (r *ResultSetMetadata) GetColumnTypeById(index int) (*ColumnType, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Type, nil
		}
	}
	return nil, errors.Errorf("failed to find column type by id")
}

func (r *ResultSetMetadata) GetColumnPrecisionById(index int) (int, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Precision, nil
		}
	}
	return -1, errors.Errorf("failed to find column precision by id")
}

func (r *ResultSetMetadata) GetColumnScaleById(index int) (int, error) {
	for _, c := range r.Columns {
		if c.Index == index {
			return c.Scale, nil
		}
	}
	return -1, errors.Errorf("failed to find column scale by id")
}

/*func (rs *ResultSetMetadata) Read(r io.Reader) error {
	size, err := ReadPackedInt(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set metadata length")
	}
}*/

func (rs *ResultSetMetadata) NewColumnOriginFromReaderWithDefault(r io.Reader, columnName string) (*ColumnOrigin, error) {
	var colOrg ColumnOrigin
	
	cn, err := TryReadPackedNil(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to peek result set column origin name as nil")
	}
	if cn == true {
		colOrg.Column = columnName
	} else {
		colOrg.Column, err = ReadPackedString(r)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column origin name")
		}
	}
	
	schemaNameIdx, err := TryReadPackedIntWithDefault(r, int64(-1))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to peek result set column origin schema name index as int")
	}
	if schemaNameIdx != -1 {
		// First skip int value
		// Get schema name from previous columns with index
		colOrg.Schema, err = rs.GetSchemaOriginNameById(int(schemaNameIdx))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column origin schema name from index")
		}
	} else {
		colOrg.Schema, err = ReadPackedString(r)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column origin schema name")
		}
	}
	
	tableNameIdx, err := TryReadPackedIntWithDefault(r, int64(-1))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to peek result set column origin table name index as int")
	}
	if tableNameIdx != -1 {
		// Get table name from previous columns with index
		colOrg.Table, err = rs.GetTableOriginNameById(int(tableNameIdx))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column origin table name from index")
		}
	} else {
		colOrg.Table, err = ReadPackedString(r)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column origin table name")
		}
	}
	
	if Debug { fmt.Printf("DEBUG: in NewColumnOriginFromReaderWithDefault, colOrg = %s\n", colOrg.ToJsonString()) }
	
	return &colOrg, nil
}

func (rs *ResultSetMetadata) NewColumnMetadataFromReader(r io.Reader, index int) error {
	var colMet ColumnMetadata

	// Property count
	propCnt, err := ReadPackedInt(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column property count")
	}
	//fmt.Printf("DEBUG: in NewColumnMetadataFromReader, propCnt = %d\n", propCnt)

	//assert propCnt >= 6;

	colMet.Index = index

	colMet.Name, err = ReadPackedString(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column name")
	}

	colMet.Nullable, err = ReadPackedBool(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column is nullable")
	}

	ct, err := ReadPackedInt(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column type")
	}
	colMet.Type, err = ColTypes.GetColumnTypeById(int(ct))
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column type kind (column name %s, id = %d)", colMet.Name, ct)
	}

	cs, err := ReadPackedInt(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column scale")
	}
	// Do UNDEFINED_SCALE = 0x80000000 ? How should we handle this value?
	colMet.Scale = int(cs)

	cp, err := ReadPackedInt(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column precision")
	}
	colMet.Precision = int(cp)

	hasOrigin, err := ReadPackedBool(r)
	if err != nil {
		return errors.Wrapf(err, "failed to read result set column origin")
	} else {
		if hasOrigin {
			// assert propCnt >= 9;
			if propCnt < 9 {
				return errors.Errorf("not enough properties to read column origin")
			}
			colMet.Origin, err = rs.NewColumnOriginFromReaderWithDefault(r, colMet.Name)
		} else {
			colMet.Origin = nil
		}
		if err != nil {
			return errors.Wrapf(err, "failed to read result set column origin")
		}
	}
	
	if Debug { fmt.Printf("DEBUG: in NewColumnMetadataFromReader, colMet = %s\n", colMet.ToJsonString()) }

	rs.Columns = append(rs.Columns, colMet)

	return nil
}

func NewResultSetMetadataFromReader(r io.Reader) (*ResultSetMetadata, error) {
	var rs ResultSetMetadata
	columnCount, err := ReadPackedInt(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read result set metadata length")
	}

	for i := 0 ; i < int(columnCount) ; i++ {
		err := rs.NewColumnMetadataFromReader(r, i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read result set column metadata")
		}
	}

	return &rs, nil
}
