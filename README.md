# Binary tuple (key-value) write path

This document describes the key-value write API added to `ignite3-go-client`: how a row is
encoded, which operations are used, what has been verified against a live cluster, and what
limitations currently exist.

The KV path exists because `MERGE` is a poor fit for high-volume CDC ingestion. On Ignite 3.1
`EXPLAIN` shows `MERGE ... USING (VALUES ...)` plans as a **hash join over a full table scan**,
so the cost of every statement grows with the table, and each distinct statement text occupies
an entry in the server plan cache (the default `ignite.sql.planner.estimatedNumberOfQueries` is
1024). The KV path sends rows as binary tuples through `OP_TUPLE_UPSERT_ALL` / `OP_TUPLE_DELETE_ALL`
instead: a plain key-based write with no parsing, no planning, no plan cache entries and no scan.

Measured on a 36-column table: a full replay of ~135 000 rows completed in ~2 minutes with zero
consumer lag, where the previous `MERGE`-based writer had degraded to planner timeouts and an
out-of-memory kill.

---

## 1. Quick start

```go
import (
    ignite3 "github.com/GHKyle/ignite3-go-client/binary/v1"
    _ "github.com/GHKyle/ignite3-go-client/sql" // only needed for the database/sql driver
)

cli, err := ignite3.Connect(ignite3.ConnInfo{
    Network: "tcp", Host: "127.0.0.1", Port: 10800, Major: 3,
})
if err != nil {
    return err
}
defer cli.Close()

// Resolve the table once. The name is a qualified name; quoting and case are flexible.
tbl, err := cli.GetTableByName(`"MY_SCHEMA"."MY_TABLE"`)
if err != nil {
    return err
}

// Upsert: insert rows, or overwrite them when the primary key already exists.
// One network round trip for the whole slice.
err = tbl.UpsertAll([]map[string]interface{}{
    {"ID": "k1", "NAME": "first",  "TS": time.Now(), "AMOUNT": ignite3.Decimal{Unscaled: big.NewInt(100050), Scale: 2}},
    {"ID": "k2", "NAME": nil,      "TS": nil,        "AMOUNT": nil}, // NULLs are fine
})

// Delete by key (only the key columns are encoded).
err = tbl.DeleteAll([]map[string]interface{}{
    {"ID": "k3"},
})
```

Keys of the row maps are case-insensitive. Columns that are missing from the map are written as
`NULL` (this is distinct from "column absent", see §5).

Low-level usage, if you already know the schema:

```go
raw, err := ignite3.BuildBinaryTuple(
    []ignite3.BinaryTupleColumn{
        {Kind: ignite3.BtString},
        {Kind: ignite3.BtDecimal, Scale: 2},
        {Kind: ignite3.BtDateTime},
        {Kind: ignite3.BtBoolean},
    },
    []interface{}{"k1", "1000.50", "2026-04-20 16:29:40.644838", true},
)
```

---

## 2. What was added

| File | Contents |
| --- | --- |
| `binary/v1/binary-tuple.go` | `BuildBinaryTuple`, `BinaryTupleColumn`, the `Bt*` element kinds and all per-type encoders |
| `binary/v1/client-schemas.go` | `Table`, `TableSchema`, `TableColumn`, `GetTableSchema` (`OP_SCHEMAS_GET`), `GetTableByName` (`OP_TABLE_GET` + `OP_TABLES_GET` fallback) |
| `binary/v1/client-tuple-operations.go` | `TableUpsertAll` / `TableDeleteAll` (`OP_TUPLE_UPSERT_ALL` = 13, `OP_TUPLE_DELETE_ALL` = 29), `Table.UpsertAll` / `Table.DeleteAll`, `Table.SchemaVersion` |
| `binary/v1/client.go` | The four methods above added to the `Client` interface |
| `binary/v1/binary-tuple_test.go` | Unit tests for the encoder layout |
| `binary/v1/table-kv-live_test.go` | End-to-end test against a live cluster, skipped unless `IGNITE_TEST_HOST` is set |

`Table` methods:

```go
type Table struct {
    Id     int            // table id, sent with every tuple operation
    Name   string         // name as reported by the cluster
    Schema *TableSchema   // Version + Columns (schema order, with Kind/Scale per column)
    // contains filtered or unexported fields
}

func (t *Table) SchemaVersion() int
func (t *Table) UpsertAll(rows []map[string]interface{}) error
func (t *Table) DeleteAll(keys []map[string]interface{}) error

func (s *TableSchema) KeyColumns() []TableColumn // key columns ordered by key index
```

Running the live test:

```bash
IGNITE_TEST_HOST=127.0.0.1:10800 go test -run TestTableKvLive -v ./binary/v1/
```

---

## 3. Wire format

### 3.1 One row: the binary tuple (IEP-92)

```
+--------+------------------+------------------+
| header |   offset table   |    value area    |
+--------+------------------+------------------+
   1 byte     entrySize*N       concatenated values
```

* **header** — one byte; its two low bits hold `log2(entrySize)`, so the offset table uses
  1, 2 or 4 bytes per entry depending on the size of the value area (`<= 255`, `<= 65535`,
  otherwise 4).
* **offset table** — one entry per column, little-endian: the **end** offset of that column
  inside the value area. Entry `i` starts where entry `i-1` ended.
* **value area** — the encoded values, in schema order.

The important consequence: **a NULL is a zero-length element**, so its end offset equals the
previous element's. Variable-length types (`VARCHAR`, `VARBINARY`) therefore encode an *empty*
value with a leading `0x80` byte, which is what makes zero length unambiguous.

### 3.2 A real example

Columns from a live 36-column table (`VARCHAR` primary key, `DECIMAL(30,2)`,
`TIMESTAMP(6)`, `BOOLEAN`, nullable `VARCHAR`):

| column | value |
| --- | --- |
| `MT4_ACCOUNT_ID` | `3f2b1c9e-8a4d-4f1e-9c2b-77aa11bb22cc` |
| `BK_OPENNING_AMOUNT` | `1000.50` |
| `OI_REGISTERED_DATETIME` | `2026-04-20 16:29:40.644838` |
| `TRADABLE` | `true` |
| `CM_PROCESS_REMARK` | `NULL` |

Encoded tuple (57 bytes):

```
00                                  header: entrySize = 1
24 29 32 33 33                      offset table: 36, 41, 50, 51, 51
33 66 32 62 ...                     [0] 36 B  STRING   "3f2b1c9e-..."
02 00 01 86 d2                      [1]  5 B  DECIMAL  scale=2, unscaled=0x0186D2=100050 -> 1000.50
94 d4 0f  70 72 6f 26 da 41         [2]  9 B  DATETIME date(3 B) + time(6 B)
01                                  [3]  1 B  BOOLEAN  true
                                    [4]  0 B  NULL (end offset still 51)
```

Decoding it by hand:

* `DECIMAL` — `int16` scale little-endian (`0x0002`) followed by the **big-endian
  two's-complement** unscaled value (`0x0186D2` = 100 050) → `1000.50`.
* `DATE` — 3 bytes little-endian, `day = v & 0x1F`, `month = (v >> 5) & 0xF`,
  `year = v >> 9` (15-bit signed): `0x0FD494` → 2026-04-20.
* `TIME` / `DATETIME` — the time part packs the fraction (30 bits), second (6), minute (6) and
  hour (5) little-endian: `0x41DA266F7270` → 644 838 000 ns, 40 s, 29 min, 16 h.

### 3.3 Encoding rules (all bytes below are measured output)

| value | bytes | rule |
| --- | --- | --- |
| `NULL` | *(0 bytes)* | zero-length element |
| `""` | `80` | empty string, distinguishable from NULL |
| `"abc"` | `61 62 63` | raw UTF-8 |
| `BOOLEAN false` / `true` | `00` / `01` | |
| `INT 5` | `05` | integers use the narrowest width that fits |
| `INT 300` | `2c 01` | does not fit in a byte -> short |
| `INT -1` | `ff` | |
| `BIGINT 70000` | `70 11 01 00` | BIGINT order is short, int, long |
| `DECIMAL(20,2) 1000.50` | `02 00 01 86 d2` | scale (LE) + big-endian two's complement |
| `DECIMAL(20,2) -1000.50` | `02 00 fe 79 2e` | |
| `DATE 2026-04-20` | `94 d4 0f` | |
| `TIMESTAMP 2026-04-20 16:29:40.644838` | `94 d4 0f 70 72 6f 26 da 41` | date + time |
| `DOUBLE` representable as float32 | 4 bytes | falls back to float when lossless |
| `VARBINARY {01 80 ff}` | `01 80 ff` | `0x80` prefix only when empty or already starting with `0x80` |

### 3.4 The request

`OP_TUPLE_UPSERT_ALL` payload, exactly as produced by `client.tupleWriteAll`
(`schemaVersion` and the row count are msgpack integers, tuples are msgpack binary):

```
int   tableId
nil                       // no transaction
int   schemaVersion
int   rowCount
rowCount times:
    c7 00 08              // "no value" bit set: msgpack ext8, length 0, type 8 (BITMASK)
    binary(tupleBytes)    // e.g. c4 39 <57 bytes>  (msgpack bin8, length 0x39)
```

The bit set marks columns that are **absent** from the tuple (not columns set to NULL); this API
always sends every column, so it is empty — but it must be written, because the server expects a
msgpack extension there and rejects a nil.

`OP_TUPLE_DELETE_ALL` has the same payload with tuples built from the **key columns only**.

---

## 4. Type mapping

`TableColumn.Kind` is derived from the protocol column type id by `columnKindForTypeId`:

| type id | SQL type | binary tuple kind |
| --- | --- | --- |
| 1 | `BOOLEAN` | `BtBoolean` |
| 2 / 3 / 4 / 5 | `TINYINT` / `SMALLINT` / `INT` / `BIGINT` | `BtInt8` / `BtInt16` / `BtInt32` / `BtInt64` |
| 6 / 7 | `REAL` / `DOUBLE` | `BtFloat` / `BtDouble` |
| 8 | `DECIMAL(p,s)` | `BtDecimal` (uses the column scale) |
| 9 / 10 | `DATE` / `TIME` | `BtDate` / `BtTime` |
| 11 | `TIMESTAMP` | `BtDateTime` (date + time, no time zone) |
| 12 | `TIMESTAMP WITH LOCAL TIME ZONE` | `BtTimestamp` (epoch seconds, + nanoseconds when non-zero) |
| 13 | `UUID` | `BtUuid` |
| 15 | `VARCHAR` | `BtString` |
| 16 | `VARBINARY` | `BtBytes` |
| other | — | `BtNull` |

Note that in Ignite 3 a plain SQL `TIMESTAMP` column is protocol type 11, i.e. the **DATETIME**
binary tuple kind — it carries no time zone and is encoded as date + time, **not** as epoch
seconds. `TIMESTAMP WITH LOCAL TIME ZONE` is the type that maps to `BtTimestamp`.

Values may be passed as Go native types or as their string form: the encoders accept
`string`, `[]byte`, `bool`, `int*`/`uint*`, `float32/64`, `ignite3.Decimal`, `ignite3.Date`,
`ignite3.Time`, `ignite3.DateTime`, `ignite3.Timestamp`, `time.Time` and `ignite3.Uuid`, and
parse the usual timestamp layouts (`RFC3339Nano`, `2006-01-02 15:04:05[.999999999]`,
`2006-01-02`, `15:04:05[.999999999]`).

---

## 5. NULL, empty values and error handling

* A `nil` Go value, or a column missing from the row map, is written as **NULL**.
* An empty string / empty byte slice is written as the empty value, **not** as NULL (`0x80`).
* A value that cannot be converted to the column kind is **not** silently dropped: the encoder
  returns an error and `UpsertAll`/`DeleteAll` return it, so a batch is never partially written
  by a malformed value (the tuple is built for the whole batch before anything is sent).
* Writes are idempotent: `OP_TUPLE_UPSERT_ALL` overwrites by primary key, so replaying a batch
  after a retry produces the same result. This is what makes at-least-once delivery safe.

---

## 6. Verified behaviour

Checked against a live single-node Ignite 3.1.0 cluster through the Go client:

* `TestTableKvLive` — upsert overwrite semantics, NULL round trip, delete by key (self-cleaning
  temporary schema, gated on `IGNITE_TEST_HOST`).
* Value-level round trip of a KV-written row, read back through the SQL driver and compared
  column by column: `VARCHAR`, `DECIMAL(20,2)` (positive and negative), `BOOLEAN`, `INT`,
  `VARBINARY`, `DATE`, `TIME` and `TIMESTAMP(6)` all come back with the values that were written.
* NULLs of every type read back as NULL (not as `0`, `""` or a zero date).
* Empty string versus NULL is preserved **in storage** — verified with server-side evaluation
  (`LENGTH(col)` = 0 and `col IS NULL` = FALSE for an empty string; `LENGTH` = NULL and
  `IS NULL` = TRUE for a NULL), which bypasses the client decoder.
* Sub-second precision is preserved **in storage**: microsecond values survive the KV encoding
  (verified with `WHERE ts = TIMESTAMP '...644838'`, which the server evaluates).
* Bulk write throughput: a ~135 000 row replay in ~2 minutes with the consumer caught up
  (`lag = 0`).

---

## 7. Known limitations

These are in the **SQL read path** (`database/sql` / `OP_QUERY_SQL_FIELDS`), not in the KV write
path, and they are pre-existing:

1. **A full-table projection returns no rows.** `SELECT "ID" FROM t` (also with `LIMIT 1` or
   `ORDER BY`) yields zero rows and no error, while `SELECT COUNT(*) FROM t` returns the correct
   count and `SELECT "ID" FROM t WHERE "ID" = 'k'` returns the row. Filtered aggregates are not
   reliable either (the same `SELECT COUNT(*) ... WHERE pk = 'k'` has been observed returning both
   0 and 1 for a row that is present).
   Until this is fixed, verify writes with `COUNT(*)` over the whole table or with single-row
   primary-key lookups rather than with scans or filtered aggregates.
2. **`TIME`/`TIMESTAMP` values are decoded with millisecond precision.** A stored value of
   `...16:29:40.644838` is read back as `...16:29:40.644`. The value in storage is correct (see
   §6); the precision is lost while decoding the result.
3. **An empty `VARCHAR` is decoded as NULL.** `Tuple.GetValue` treats a zero-length payload as
   NULL, but the decode step has already stripped the `0x80` marker of an empty string, so the
   two become indistinguishable on the read path. Storage is unaffected.
4. `GetTableByName` works best for tables in a non-default schema: `OP_TABLE_GET` is authoritative,
   while `OP_TABLES_GET` (used as a fallback) does not necessarily list every table.
5. A table that is dropped and re-created gets a new table id and possibly a new schema version.
   Re-resolve the `Table` (or at least `GetTableSchema`) after any DDL, otherwise the write is
   sent with a stale id/version.

---

## 8. References

* https://github.com/yo000/ignite3-go-client
* IEP-92 (binary tuple format): <https://cwiki.apache.org/confluence/display/IGNITE/IEP-92+Binary+Tuple+Format>
* Java counterpart: `org.apache.ignite.internal.binarytuple.BinaryTupleBuilder`
* Ignite client protocol operations: `ClientOperation#TUPLE_UPSERT_ALL` (13),
  `ClientOperation#TUPLE_DELETE_ALL` (29)
