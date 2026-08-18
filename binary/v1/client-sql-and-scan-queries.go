package ignite3

import (
	"fmt"
	"time"
	"bytes"
	"reflect"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
)

const (
	DEFAULT_SCHEMA = "PUBLIC"
)

var (
	// TODO: Are we?
	PartitionAwareness bool

	Debug = false
)

// QuerySQLData input parameter for QuerySQL func
type QuerySQLData struct {
	// Schema for the query
	Schema string

	// SQL query string.
	Query string

	// Query arguments.
	QueryArgs []interface{}

	// Cursor page size.
	PageSize int

	// Timeout(milliseconds) value should be non-negative. Zero value disables timeout.
	Timeout int64
}

// QuerySQLPage is query result page
type QuerySQLPage struct {
	// Key -> Values
	Rows map[int]interface{}

	// Indicates whether more results are available to be fetched with QuerySQLCursorGetPage.
	// When false, query cursor is closed automatically.
	HasMore bool
}

// QuerySQLResult output from QuerySQL func
type QuerySQLResult struct {
	// Cursor id. Can be closed with ResourceClose func.
	ResponseId int64
	ResourceId int64
	Metadata   *ResultSetMetadata
	
	HasRowSet    bool
	wasApplied   bool
	AffectedRows int64
	// Query result first page
	QuerySQLPage
}

// QuerySQLFieldsData input parameter for QuerySQLFields func
type QuerySQLFieldsData struct {
	// Schema for the query; can be empty, in which case default PUBLIC schema will be used.
	Schema string

	// Query cursor page size.
	PageSize int

	// SQL query string.
	Query string

	// Query arguments.
	QueryArgs []interface{}

	// Timeout(milliseconds) value should be non-negative. Zero value disables timeout.
	Timeout int64
}

// QueryScanData input parameter for QueryScan func
type QueryScanData struct {
	// Cursor page size.
	PageSize int

	// Number of partitions to query (negative to query entire cache).
	Partitions int

	// Local flag - whether this query should be executed on local node only.
	LocalQuery bool
}



func ReadQuerySQLResponseHeader(result QuerySQLResult, response *ResponseOperation) (QuerySQLResult, error) {
	var err error
	// ClientAsyncResultSet.java:97
	var resrcId int64
	var resrcIdIsNil bool
	if resrcIdIsNil, err = TryReadPackedNil(response.message); err != nil {
		return result, errors.Wrapf(err, "failed to read resource id existence")
	}
	if !resrcIdIsNil {
		if resrcId, err = ReadPackedInt64(response.message); err != nil {
			return result, errors.Wrapf(err, "failed to read resource id value")
		}
		result.ResourceId = resrcId
	} else { result.ResourceId = 0 }
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : resource id = %x\n", result.ResourceId) }

	if result.HasRowSet, err = ReadPackedBool(response.message); err != nil {
		fmt.Printf("coincoin 3\n")
		return result, errors.Wrapf(err, "failed to read has row set")
	}
	// REMOVE ME
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : has row set = %t\n", result.HasRowSet) }
	
	if result.HasMore, err = ReadPackedBool(response.message); err != nil {
		return result, errors.Wrapf(err, "failed to read has more pages")
	}
	// REMOVE ME
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : has more pages = %t\n", result.HasMore) }

	if result.wasApplied, err = ReadPackedBool(response.message); err != nil {
		return result, errors.Wrapf(err, "failed to read was applied")
	}
	// REMOVE ME
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : was applied = %t\n", result.wasApplied) }

	if result.AffectedRows, err = ReadPackedInt64(response.message); err != nil {
		return result, errors.Wrapf(err, "failed to read affected rows qty")
	}
	// REMOVE ME
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : affected rows = %x\n", result.AffectedRows) }
	
	return result, nil
}

func ReadPartitionAwareness(response *ResponseOperation, partAwareness bool, sqlDMSupported bool) (*ClientPartitionAwarenessMetadata, error) {
	var err error
	var rsPartAwareMetadata *ClientPartitionAwarenessMetadata

	if partAwareness {
		rsPartAwareMetadata, err = NewClientPartitionAwarenessMetadataFromReader(response.message, sqlDMSupported)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read client partition awareness metadata")
		}
	} else {
		rsPartAwareMetadata = nil
	}

	return rsPartAwareMetadata, nil
}

// References : ClientSql.java, ignite3 sources
func (c *client) QuerySQL(query QuerySQLData) (QuerySQLResult, error) {
	var queryRes QuerySQLResult
	var bufr bytes.Buffer
	var err error
	
	// FIXME
	partitionAwareness := false
	// FIXME
	sqlDirectMappingSupported := false
	
	// Build operation specific data
	// Transaction ID (from ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode session id")
	}

	// FIXME: Should be default
	if len(query.Schema) == 0 { query.Schema = DEFAULT_SCHEMA }
	if err := WritePackedString(&bufr, query.Schema); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode schema")
	}
	
	// page Size (ClientSql.java)
	if err := WritePackedInt64(&bufr, int64(query.PageSize)); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode pagesize")
	}
	
	// query timeout in millisecond (ClientSql.java)
	if err := WritePackedInt64(&bufr, int64(query.Timeout)); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode query timeout")
	}
	
	// TODO : Session Timeout (ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode session timeout")
	}
	
	//  TODO : w.out().packString(statement.timeZoneId().getId()); (ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode query timezone id")
	}

	// TODO : packProperties(w, null); (ClientSql.java)
	if err := WritePackedInt64(&bufr, int64(0)); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode ?? zero value")
	}

	// TODO : packProperties(w, null); 2nd part ??
	if err := WritePackedBytes(&bufr, []byte{0}); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode marker zero value")
	}

	// Write SQL Request
	if err := WritePackedString(&bufr, query.Query); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode query string")
	}

	if len(query.QueryArgs) > 0 {
		if err := WritePackedInt64(&bufr, int64(len(query.QueryArgs))); err != nil {
			return QuerySQLResult{}, errors.Wrapf(err, "failed to encode args count")
		}
		t := NewTuple()
		for i, v := range query.QueryArgs {
			if err := t.AddValue(v); err != nil {
				return QuerySQLResult{}, errors.Wrapf(err, "failed to encode args %d", i)
			}
		}
		tupleBytes, err := t.Bytes()
		if err != nil {
			return QuerySQLResult{}, errors.Wrapf(err, "failed to encode args tuple to binary")
		}
		if err := WritePackedBytes(&bufr, tupleBytes); err != nil {
			return QuerySQLResult{}, errors.Wrapf(err, "failed to encode args tuple")
		}
	} else {
		if err := WritePackedNull(&bufr); err != nil {
			return QuerySQLResult{}, errors.Wrapf(err, "failed to encode query args")
		}
	}

	// observable timestamp (ClientSql.java)
	if err := WritePackedTimestamp(&bufr, time.Now()); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to encode observable timestamp")
	}

	// request and response
	req := NewRequestOperation(OpSqlExecute, c.RequestId, bufr.Bytes())
	res := NewResponseOperation(req.RequestId)

	// execute operation
	if err := c.Do(req, res); err != nil {
		return QuerySQLResult{}, errors.Wrapf(err, "failed to execute SQL_EXECUTE operation")
	}
	if err := res.CheckStatus(); err != nil {
		return QuerySQLResult{}, err
	}

	queryRes.ResponseId = res.RequestId

	if queryRes, err = ReadQuerySQLResponseHeader(queryRes, res); err != nil {
		return QuerySQLResult{}, err
	}

	/* Read ResultSet MetaData */
	rsMetadata, err := NewResultSetMetadataFromReader(res.message)
	if err != nil {
		fmt.Printf("ERREUR : %v\n", err)
		return QuerySQLResult{}, err
	}
	if Debug { fmt.Printf("DEBUG: rsMetadata = %v\n", rsMetadata) }
	queryRes.Metadata = rsMetadata

	var rsPartAwareMetadata *ClientPartitionAwarenessMetadata
	if rsPartAwareMetadata, err = ReadPartitionAwareness(res, partitionAwareness, sqlDirectMappingSupported); err != nil {
		return QuerySQLResult{}, err
	}
	if Debug { fmt.Printf("DEBUG: rsPartAwareMetadata = %v\n", rsPartAwareMetadata) }

	/* Read ResultSet Data */
	if (queryRes.HasRowSet) {
		queryRes.Rows = make(map[int]interface{})
		rowCount, err := ReadPackedInt64(res.message)
		if err != nil {
			return QuerySQLResult{}, errors.Wrapf(err, "failed to read row count")
		}
		//fmt.Printf("DEBUG: row count = %d\n", rowCount)
		
		/* From https://cwiki.apache.org/confluence/display/IGNITE/IEP-76%3A+Thin+Client+Protocol+for+Ignite+3.0 :
		 * thanks to schema-first approach, we can avoid sending column names with the values (serializing strings is expensive). Instead, we can write an integer schema version, and then values for every column in that schema.
		 */
		var bytes []byte
		var tuple *Tuple
		var value interface{}
		var colType *ColumnType
		var colName string
		for i := 0 ; i < int(rowCount) ; i++ {
			row := make(map[string]interface{})

			if bytes, err = ReadPackedBytes(res.message); err != nil {
				return QuerySQLResult{}, errors.Wrapf(err, "failed to read row tuple")
			}
			if tuple, err = NewTupleFromByteArrayWithMetadata(queryRes.Metadata, bytes); err != nil {
				return QuerySQLResult{}, errors.Wrapf(err, "failed to parse row tuple")
			}
			for j, _ := range queryRes.Metadata.Columns {
				if colType, err = queryRes.Metadata.GetColumnTypeById(j); err != nil {
					return QuerySQLResult{}, errors.Wrapf(err, "failed to get column type")
				}
				if colName, err = queryRes.Metadata.GetColumnOriginNameById(j); err != nil {
					return QuerySQLResult{}, errors.Wrapf(err, "failed to get column name")
				}
				if value, err = tuple.GetValue(j, colType.TypeId); err != nil {
					return QuerySQLResult{}, errors.Wrapf(err, "failed to parse column %d", j)
				}
				// Convert value if needed
				switch colType.GoKind {
					case reflect.Struct:
						switch colType.SQLTypeName {
							case "DECIMAL":
								value = value.(Decimal).String()
							case "UUID":
								value = value.(Uuid).String()
							case "DATETIME", "TIMESTAMP":
								value = value.(time.Time).String()
							default:
								return QuerySQLResult{}, errors.Errorf("Unsupported column type: %s at %d", colType.SQLTypeName, j)
						}
				}
				row[colName] = value
			}
			queryRes.Rows[i] = row
		}
	}

	return queryRes, nil
}

// QuerySQLCursorGetPage retrieves the next SQL query cursor page by cursor id from QuerySQL.
func (c *client) QuerySQLCursorGetPage(id int64, metadata *ResultSetMetadata) (QuerySQLResult, error) {
	// FIXME: Devrait être une QuerySQLPage
	var r QuerySQLResult
	r.ResourceId = id

	var bufr bytes.Buffer

	// Write requested cursor ID
	if err := WritePackedInt64(&bufr, id); err != nil {
		return r, errors.Wrapf(err, "failed to encode pagesize")
	}

	// request and response
	req := NewRequestOperation(OpSqlCursorNextPage, c.RequestId, bufr.Bytes())
	res := NewResponseOperation(req.RequestId)

	// execute operation
	if err := c.Do(req, res); err != nil {
		return r, errors.Wrapf(err, "failed to execute SQL_CURSOR_NEXT_PAGE operation")
	}
	if err := res.CheckStatus(); err != nil {
		return r, err
	}

	/* Read ResultSet Data */
	r.Rows = make(map[int]interface{})
	rowCount, err := ReadPackedInt64(res.message)
	if err != nil {
		return r, errors.Wrapf(err, "failed to read row count")
	}
	if Debug { fmt.Printf("DEBUG: row count = %d\n", rowCount) }

	/* From https://cwiki.apache.org/confluence/display/IGNITE/IEP-76%3A+Thin+Client+Protocol+for+Ignite+3.0 :
	* thanks to schema-first approach, we can avoid sending column names with the values (serializing strings is expensive). Instead, we can write an integer schema version, and then values for every column in that schema.
	*/
	var bytes[]byte
	var tuple *Tuple
	var value interface{}
	var colType *ColumnType
	var colName string
	for i := 0 ; i < int(rowCount) ; i++ {
		row := make(map[string]interface{})

		if bytes, err = ReadPackedBytes(res.message); err != nil {
			return r, errors.Wrapf(err, "failed to read row tuple")
		}
		if tuple, err = NewTupleFromByteArrayWithMetadata(metadata, bytes); err != nil {
			return r, errors.Wrapf(err, "failed to parse row tuple")
		}
		for j, _ := range metadata.Columns {
			if colType, err = metadata.GetColumnTypeById(j); err != nil {
				return r, errors.Wrapf(err, "failed to get column type")
			}
			if colName, err = metadata.GetColumnOriginNameById(j); err != nil {
				return r, errors.Wrapf(err, "failed to get column name")
			}
			if value, err = tuple.GetValue(j, colType.TypeId); err != nil {
				return r, errors.Wrapf(err, "failed to parse column %d", j)
			}
			// Convert value if needed
			switch colType.GoKind {
				case reflect.Struct:
					switch colType.SQLTypeName {
						case "DECIMAL":
							value = value.(Decimal).String()
						case "UUID":
							value = value.(Uuid).String()
						case "DATE", "TIME", "DATETIME", "TIMESTAMP":
							value = value.(time.Time).String()
						default:
							return QuerySQLResult{}, errors.Errorf("Unsupported column type: %s at %d", colType.SQLTypeName, j)
					}
			}
			row[colName] = value
		}
		r.Rows[i] = row
	}

	hasMorePages, err:= ReadPackedBool(res.message)
	if err != nil {
		return r, errors.Wrapf(err, "failed to read has more pages")
	}
	r.HasMore = hasMorePages
	if Debug { fmt.Printf("DEBUG in (c *client) QuerySQL : has more pages = %t\n", hasMorePages) }

	return r, nil
}

func (c *client) QuerySQLFieldsRaw(data QuerySQLFieldsData) (*ResponseOperation, error) {
	var bufr bytes.Buffer

	// Build operation specific data
	// Transaction ID (from ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return nil, errors.Wrapf(err, "failed to encode transaction id")
	}

	// FIXME: Should be default
	if len(data.Schema) == 0 { data.Schema = DEFAULT_SCHEMA }
	if err := WritePackedString(&bufr, data.Schema); err != nil {
		return nil, errors.Wrapf(err, "failed to encode schema")
	}

	if err := WritePackedInt64(&bufr, int64(data.PageSize)); err != nil {
		return nil, errors.Wrapf(err, "failed to encode pagesize")
	}

	if err := WritePackedInt64(&bufr, int64(data.Timeout)); err != nil {
		return nil, errors.Wrapf(err, "failed to encode query timeout")
	}

	// TODO : Session Timeout (ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return nil, errors.Wrapf(err, "failed to encode session timeout")
	}

	//  TODO : w.out().packString(statement.timeZoneId().getId()); (ClientSql.java)
	if err := WritePackedNull(&bufr); err != nil {
		return nil, errors.Wrapf(err, "failed to encode query id")
	}

	// TODO : packProperties(w, null); (ClientSql.java)
	if err := WritePackedInt64(&bufr, int64(0)); err != nil {
		return nil, errors.Wrapf(err, "failed to encode ?? zero value")
	}

	// TODO : packProperties(w, null); 2nd part ??
	if err := WritePackedBytes(&bufr, []byte{0}); err != nil {
		return nil, errors.Wrapf(err, "failed to encode marker zero value")
	}

	// Write SQL Request
	if err := WritePackedString(&bufr, data.Query); err != nil {
		return nil, errors.Wrapf(err, "failed to encode query string")
	}

	if len(data.QueryArgs) > 0 {
		if err := WritePackedInt64(&bufr, int64(len(data.QueryArgs))); err != nil {
			return nil, errors.Wrapf(err, "failed to encode args count")
		}
		t := NewTuple()
		for i, v := range data.QueryArgs {
			if err := t.AddValue(v); err != nil {
				return nil, errors.Wrapf(err, "failed to encode args %d", i)
			}
		}
		tupleBytes, err := t.Bytes()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to encode args tuple to binary")
		}
		if err := WritePackedBytes(&bufr, tupleBytes); err != nil {
			return nil, errors.Wrapf(err, "failed to encode args tuple")
		}
		
	} else {
		if err := WritePackedNull(&bufr); err != nil {
			return nil, errors.Wrapf(err, "failed to encode query args")
		}
	}

	if err := WritePackedTimestamp(&bufr, time.Now()); err != nil {
		return nil, errors.Wrapf(err, "failed to encode observable timestamp")
	}

	// request and response
	req := NewRequestOperation(OpSqlExecute, c.RequestId, bufr.Bytes())
	res := NewResponseOperation(req.RequestId)

	// execute operation
	if err := c.Do(req, res); err != nil {
		return nil, errors.Wrapf(err, "failed to execute SQL_EXECUTE operation")
	}
	if err := res.CheckStatus(); err != nil {
		return nil, err
	}

	return res, nil
}

// QuerySQLFields performs SQL fields query.
func (c *client) QuerySQLFields(data QuerySQLFieldsData) (QuerySQLResult, error) {
	var r QuerySQLResult
	var err error

	// FIXME
	partitionAwareness := false
	// FIXME
	sqlDirectMappingSupported := false
	
	var res *ResponseOperation
	if res, err = c.QuerySQLFieldsRaw(data); err != nil {
		return r, err
	}

	r.ResponseId = res.RequestId

	// Read resource Id, hasRowSet, hasMore, wasApplied, AffectedRows
	if r, err = ReadQuerySQLResponseHeader(r, res); err != nil {
		return r, err
	}

	/* Read ResultSet MetaData */
	if r.Metadata, err = NewResultSetMetadataFromReader(res.message); err != nil {
		fmt.Printf("ERREUR : %v\n", err)
		return r, err
	}
	if Debug { fmt.Printf("DEBUG: rsMetadata = %v\n", r.Metadata) }

	var rsPartAwareMetadata *ClientPartitionAwarenessMetadata
	if rsPartAwareMetadata, err = ReadPartitionAwareness(res, partitionAwareness, sqlDirectMappingSupported); err != nil {
		return r, err
	}
	if Debug { fmt.Printf("DEBUG: rsPartAwareMetadata = %v\n", rsPartAwareMetadata) }

	/* Read ResultSet Data */
	if (r.HasRowSet) {
		var rowCount int64
		r.Rows = make(map[int]interface{})
		if rowCount, err = ReadPackedInt64(res.message); err != nil {
			return r, errors.Wrapf(err, "failed to read row count")
		}
		//fmt.Printf("DEBUG: row count = %d\n", rowCount)
		
		/* From https://cwiki.apache.org/confluence/display/IGNITE/IEP-76%3A+Thin+Client+Protocol+for+Ignite+3.0 :
		* thanks to schema-first approach, we can avoid sending column names with the values (serializing strings is expensive). Instead, we can write an integer schema version, and then values for every column in that schema.
		*/
		var bytes []byte
		var tuple *Tuple
		var value interface{}
		var colType *ColumnType
		var colName string
		for i := 0 ; i < int(rowCount) ; i++ {
			row := make(map[string]interface{})

			if bytes, err = ReadPackedBytes(res.message); err != nil {
				return r, errors.Wrapf(err, "failed to read row tuple")
			}
			if tuple, err = NewTupleFromByteArrayWithMetadata(r.Metadata, bytes); err != nil {
				return r, errors.Wrapf(err, "failed to parse row tuple")
			}
			abortRow := false
			for j, _ := range r.Metadata.Columns {
				if colType, err = r.Metadata.GetColumnTypeById(j); err != nil {
					return r, errors.Wrapf(err, "failed to get column type parsing rows")
				}
				if colName, err = r.Metadata.GetColumnOriginNameById(j); err != nil {
					return r, errors.Wrapf(err, "failed to get column name")
				}
				if value, err = tuple.GetValue(j, colType.TypeId); err != nil {
					return r, errors.Wrapf(err, "failed to parse column %d", j)
				}
				// Convert value if needed
				switch colType.GoKind {
					case reflect.Struct:
						switch colType.SQLTypeName {
							case "DECIMAL":
								value = value.(Decimal).String()
							case "UUID":
								value = value.(Uuid).String()
							case "DATE", "TIME", "DATETIME", "TIMESTAMP":
								value = value.(time.Time).String()
							default:
								return QuerySQLResult{}, errors.Errorf("Unsupported column type: %s at %d", colType.SQLTypeName, j)
						}
				}
				row[colName] = value
			}
			if !abortRow {
				r.Rows[i] = row
			}
		}
	}

	return r, nil
}

func (c *client) ResourceClose(resourceId int64) error {
	var err error
	if resourceId == 0 {
		return nil
	}

	req := NewRequestOperation(OpSqlCursorClose, c.RequestId, nil)
	res := NewResponseOperation(req.RequestId)

	// execute operation
	if err = c.Do(req, res); err != nil {
		return errors.Wrapf(err, "failed to execute SQL_CURSOR_CLOSE operation")
	}
	if err := res.CheckStatus(); err != nil {
		return err
	}

	return nil
}
