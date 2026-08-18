# ignite3-go-client

Ignite v3 binary driver for Golang  
Based on https://github.com/yo000/ignite-go-client  

database/sql driver support, and direct queries though Connect & QuerySQL/QuerySQLFields  

## How to use client
Import client package :
```
import (
	"github.com/GHKyle/ignite3-go-client/binary/v1"
)
```

Connect to server :
```
	var c   ignite3.Client
	var err error

	if cnx, err = ignite3.Connect(ignite3.ConnInfo{
			Network:  "tcp",
			Host:     "localhost",
			Port:     10800,
			Major:    3,
			Minor:    0,
			Patch:    0,
			Username: "ignite",
			Password: "ignite",
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}); err != nil {
			fmt.Printf("Connect() error = %v\n", err)
			return
		}
```

Execute query :
```
	var res QuerySQLResult
	data := ignite3.QuerySQLFieldsData{
		Schema: sqlSchema,
		PageSize: 10240,
		MaxRows:  32768,
		Query: "SELECT TRACKID, NAME FROM TRACK WHERE NAME LIKE ?",
		QueryArgs: []interface{}{nameArg},
	}

	if res, err = c.QuerySQLFields(data); err != nil {
		fmt.Printf("Error : %v\n", err)
	}

	for i, v := range res.Rows {
		fmt.Printf("%v\n", v)
	}
```


## How to use SQLdriver
Import SQL driver :
```
import (
    "database/sql"
    _ "github.com/GHKyle/ignite3-go-client/sql"
)
```

Connect to server :
```
	ctx := context.Background()

	var db *sql.DB
	if db, err = sql.Open("ignite3", "tcp://127.0.0.1:10800/SQL_PUBLIC_ARTIST?version=3.0.0&tls=yes&tls-insecure-skip-verify=yes&page-size=10000&timeout=5000"); err != nil {
		fmt.Printf("failed to open connection: %v", err)
		return
	}
	defer db.Close()

	// Validate connection
	if err = db.PingContext(ctx); err != nil {
		fmt.Printf("ping failed: %v\n", err)
		return
	}
```

Execute query :
```
	query := "SELECT TRACKID, NAME FROM TRACK ORDER BY TRACKID"

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		fmt.Printf("failed to prepare statement: %v\n", err)
		return
	}
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		fmt.Printf("failed sql query: %v\n", err)
		return
	}
	defer rows.Close()

	var trackId int32
	var trackName string
	for rows.Next() {
		if err := rows.Scan(&trackId, &trackName); err != nil {
			fmt.Printf("failed to get row: %v", err)
			return
		}
		fmt.Printf("%d: %s\n", trackId, trackName)
	}
```


## Type mapping

| Apache Ignite Type    | Go language type                                                       | SQL Type        |
| --------------------- | ---------------------------------------------------------------------- | --------------- |
| byte                  | byte                                                                   | TINYINT         |
| short                 | int16                                                                  | SMALLINT        |
| int                   | int32                                                                  | INT             |
| long                  | int64, int                                                             | BIGINT          |
| float                 | float32                                                                | REAL            |
| double                | float64                                                                | DOUBLE          |
| bool                  | bool                                                                   | BOOLEAN         |
| String                | string                                                                 | VARCHAR         |
| UUID (Guid)           | ignite3.Uuid                                                           | UUID            |
| Date                  | ignite3.Date                                                           | DATE            |
| Decimal               | ignite3.Decimal                                                        | DECIMAL         |
| Timestamp             | ignite3.DateTime                                                       | DATETIME        |
| Timestamp w/ local TZ | ignite3.Timestamp                                                      | TIMESTAMP       |
| Time                  | ignite3.Time                                                           | TIME            |
| NULL                  | nil                                                                    | NULL            |
| byte array            | []byte                                                                 | VARBINARY       |


# Common issues
```
failed to create client: failed to open connection: EOF
```
Check your TLS settings on ignite  


# References
https://cwiki.apache.org/confluence/display/IGNITE/IEP-76%3A+Thin+Client+Protocol+for+Ignite+3.0  
https://cwiki.apache.org/confluence/display/IGNITE/IEP-92%3A+Binary+Tuple+Format  

