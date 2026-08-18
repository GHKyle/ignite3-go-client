package ignite3
/*
 * Ref : modules/platforms/cpp/ignite/protocol/client_operation.h
 */

// v3_OK
const (
	OpHeartbeat                =  1
	OpTablesGet                =  3
	OpTableGet                 =  4
	OpSchemaGet                =  5
	OpTupleUpsert              = 10
	OpTupleGet                 = 12
	OpTupleUpsertAll           = 13
	OpTupleGetAll              = 15
	OpTupleGetAndUpsert        = 16
	OpTupleInsert              = 18
	OpTupleInsertAll           = 20
	OpTupleReplace             = 22
	OpTupleReplaceExact        = 24
	OpTupleGetAndReplace       = 26
	OpTupleDelete              = 28
	OpTupleDeleteAll           = 29
	OpTupleDeleteExact         = 30
	OpTupleDeleteAllExact      = 31
	OpTupleGetAndDelete        = 32
	OpTupleContainsKey         = 33
	OpJdbcTableMeta            = 38
	OpJdbcColumnMeta           = 39
	OpJdbcPKMeta               = 41
	OpTxBegin                  = 43
	OpTxCommit                 = 44
	OpTxRollback               = 45
	OpComputeExecute           = 47
	OpClusterGetNodes          = 48
	OpComputeExecuteCollocated = 49
	OpSqlExecute               = 50
	OpSqlCursorNextPage        = 51
	OpSqlCursorClose           = 52
	OpSqlExecuteScript         = 56
	OpSqlQueryMeta             = 57
	OpComputeGetStatus         = 59
	OpComputeCancel            = 60
	OpComputeChangePriority    = 61
	OpSqlExecuteBatch          = 63
	OpSqlCancelExecute         = 70
	OpTablesGetQualified       = 71
	OpTableGetQualified        = 72
)
