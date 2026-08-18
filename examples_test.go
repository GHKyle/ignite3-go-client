package examples

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	ignite3 "github.com/GHKyle/ignite3-go-client/binary/v1"
	_ "github.com/GHKyle/ignite3-go-client/sql"
)

// Test TCP connection and GetTables
func TestConnection(t *testing.T) {
	cnx, err := ignite3.Connect(ignite3.ConnInfo{
			Network:  "tcp",
			Host:     "localhost",
			Port:     10800,
			Major:    3,
			Minor:    0,
			Patch:    0,
			Username: "ignite",
			Password: "ignite",
			TLSConfig: &tls.Config{
				// You should only set this to true for testing purposes.
				InsecureSkipVerify: true,
			},
		})
	if (err != nil) {
		t.Fatalf("Connect() error = %v", err)
		return
	}
	defer cnx.Close()

	tables, err := cnx.GetTables()
	if (err != nil) {
		t.Fatalf("GetTables() error = %v", err)
		return
	}
	for k, v := range tables {
		fmt.Printf("%s:%d\n", v, k)
	}

	fmt.Printf("Done!\n",)
}

func Test_SQL_Driver(t *testing.T) {
	ctx := context.Background()

	// open connection
	db, err := sql.Open("ignite3", "tcp://localhost:10800/Track?"+
		"version=3.0.0"+
		// Credentials are only needed if they're configured in your Ignite server.
		"&username=ignite"+
		"&password=ignite"+
		// Don't set "tls=yes" if your Ignite server
		// isn't configured with any TLS certificates.
		"&tls=yes"+
		// You should only set this to true for testing purposes.
		"&tls-insecure-skip-verify=yes"+
		"&page-size=10000"+
		"&timeout=5000")
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	defer db.Close()

	// ping
	if err = db.PingContext(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// clear test data from server
	defer func() {
		db.ExecContext(ctx, "DROP TABLE TEST_TYPES")
		log.Printf("Table TEST_TYPES dropped")
	}()

	// delete
	query := "CREATE TABLE TEST_TYPES(BOOL BOOLEAN, TINT TINYINT PRIMARY KEY, SINT SMALLINT, INT INT, BINT BIGINT," +
		"REAL REAL, DOUBLE DOUBLE, TSTAMP TIMESTAMP, LTSTAMP TIMESTAMP WITH LOCAL TIME ZONE, DAT DATE, TIM TIME," +
		"UID UUID, VARCHAR VARCHAR, VARBINARY VARBINARY);"
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		t.Fatalf("failed sql execute: %v", err)
	}
	log.Printf("Table TEST_TYPES created")

	// insert
	res, err = db.ExecContext(ctx, "INSERT INTO TEST_TYPES VALUES (true, 0, 1, 2, 3, 4.3, 5.6789, LOCALTIMESTAMP," +
					"CURRENT_TIMESTAMP, CURRENT_DATE, LOCALTIME, RAND_UUID(), 'six', x'801234')")
	if err != nil {
		t.Fatalf("failed sql execute: %v", err)
	}
	c, _ := res.RowsAffected()
	log.Printf("inserted rows: %d", c)

	// insert using prepare statement
	stmt, err := db.PrepareContext(ctx, "INSERT INTO TEST_TYPES (BOOL, TINT, SINT, INT, BINT, REAL, DOUBLE, TSTAMP," +
			"LTSTAMP, DAT, TIM, UID, VARCHAR, VARBINARY) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)," +
			"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		t.Fatalf("failed to prepare statement: %v", err)
	}
	ts1, _ := time.Parse("2006-01-02 15:04:05.999 -0700 MST", "2026-01-07 20:15:28.264 +0100 CET")
	lts1, _ := time.Parse("2006-01-02 15:04:05.999", "2026-01-10 18:13:28.477")
	dt1, _ := time.Parse("2006-01-02", "2026-01-07")
	tm1, _ := time.Parse("15:04:05", "20:15:28")
	var u1, u2 ignite3.Uuid
	u1.FromString("d8cb6cca-64ed-447b-90ef-59e9acd95193")
	u2.New()
	res, err = stmt.ExecContext(ctx,
				    bool(true), 1, 7, 8, 9, 10.1, 11, ignite3.Timestamp{Time:ts1}, 
				    ignite3.Timestamp{Time:lts1}, ignite3.Date{Time: dt1}, 
				    ignite3.Time{Time:tm1}, u1, 
				    "test", []byte{0x67,0x89,0xab,0xcd},
				    bool(false), 2, 12, 13, 14, 15.15, 16.987654321, ignite3.Timestamp{Time:ts1}, 
				    ignite3.Timestamp{Time:lts1}, ignite3.Date{Time:dt1}, 
				    ignite3.Time{Time:tm1}, u2, 
				    "", []byte{0xde,0xad,0xbe,0xef})
	if err != nil {
		t.Fatalf("failed sql execute: %v", err)
	}
	c, _ = res.RowsAffected()
	log.Printf("inserted rows: %d", c)

	// update
	res, err = db.ExecContext(ctx, "UPDATE TEST_TYPES SET TSTAMP = ? WHERE TINT = ?", ignite3.Timestamp{Time:time.Now()}, int64(2))
	if err != nil {
		t.Fatalf("failed sql execute: %v", err)
	}
	c, _ = res.RowsAffected()
	log.Printf("updated rows: %d", c)

	// select
	stmt, err = db.PrepareContext(ctx,
		"SELECT TINT, DOUBLE, TSTAMP FROM TEST_TYPES WHERE BOOL = ? ORDER BY TINT ASC")
	if err != nil {
		t.Fatalf("failed to prepare statement: %v", err)
	}
	rows, err := stmt.QueryContext(ctx, true)
	if err != nil {
		t.Fatalf("failed sql query: %v", err)
	}
	cols, _ := rows.Columns()
	log.Printf("columns: %v", cols)
	var (
		tint    int64
		double  float64
		tstamp  ignite3.Timestamp
	)
	for rows.Next() {
		if err := rows.Scan(&tint, &double, &tstamp); err != nil {
			t.Fatalf("failed to get row: %v", err)
		}
		log.Printf("tint=%d, double=\"%f\", tstamp=\"%v\"", tint, double, tstamp)
	}
}

func Test_SQL_Queries(t *testing.T) {
	// connect
	c, err := ignite3.Connect(ignite3.ConnInfo{
		Network: "tcp",
		Host:    "localhost",
		Port:    10800,
		Major:   3,
		Minor:   0,
		Patch:   0,
		// Credentials are only needed if they're configured in your Ignite server.
		Username: "ignite",
		Password: "ignite",
		Dialer: net.Dialer{
			Timeout: 10 * time.Second,
		},
		// Don't set the TLSConfig if your Ignite server
		// isn't configured with any TLS certificates.
		TLSConfig: &tls.Config{
			// You should only set this to true for testing purposes.
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		t.Fatalf("failed connect to server: %v", err)
	}
	defer c.Close()

	// clear test data from server
	defer func() {
		_, err = c.QuerySQL(ignite3.QuerySQLData{
			PageSize: 10,
			Query: "DROP TABLE TEST_TYPES",
			Timeout:           10000,
		})
		if err != nil {
			t.Fatalf("failed drop table: %v", err)
		} else {
			log.Printf("Table TEST_TYPES dropped")
		}
	}()

	// delete
	query := "CREATE TABLE TEST_TYPES(BOOL BOOLEAN, TINT TINYINT PRIMARY KEY, SINT SMALLINT, INT INT, BINT BIGINT, " +
		"REAL REAL, DOUBLE DOUBLE, TSTAMP TIMESTAMP, LTSTAMP TIMESTAMP WITH LOCAL TIME ZONE, DAT DATE, TIM TIME," +
		"UID UUID, VARCHAR VARCHAR, VARBINARY VARBINARY);"
	_, err = c.QuerySQL(ignite3.QuerySQLData{
		PageSize: 1,
		Query: query,
	})
	if err != nil {
		t.Fatalf("failed sql query: %v", err)
	} else {
		log.Printf("Table TEST_TYPES created")
	}

	// insert data
	ts1, _ := time.Parse("2006-01-02 15:04:05.999 -0700 MST", "2025-01-07 20:15:28.264 +0100 CET")
	lts1, _ := time.Parse("2006-01-02 15:04:05.999", "2025-01-10 18:13:28.477")
	dt1, _ := time.Parse("2006-01-02", "2025-01-07")
	tm1, _ := time.Parse("15:04:05", "20:15:28")
	var u1, u2 ignite3.Uuid
	u1.FromString("396e3e53-d8d1-4363-9fe6-fa4d2f284ea8")
	u2.New()

	_, err = c.QuerySQLFields(ignite3.QuerySQLFieldsData{
		PageSize: 10,
		Query: "INSERT INTO TEST_TYPES (BOOL, TINT, SINT, INT, BINT, REAL, DOUBLE, TSTAMP," +
				 " LTSTAMP, DAT, TIM, UID, VARCHAR, VARBINARY) VALUES " +
				 "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)," +
				 "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		QueryArgs: []interface{}{
			bool(true), 3, 17, 18, 19, 20.2, 21, ignite3.Timestamp{Time:ts1}, 
			ignite3.Timestamp{Time:lts1}, ignite3.Date{Time: dt1}, 
			ignite3.Time{Time:tm1}, u1, 
			"qsqlfields1", []byte{0x00,0x11,0x22,0x33},
			bool(false), 2, 12, 13, 14, 15.15, 16.987654321, ignite3.Timestamp{Time:ts1}, 
			ignite3.Timestamp{Time:lts1}, ignite3.Date{Time:dt1}, 
			ignite3.Time{Time:tm1}, u2, 
			"qsqlfields2", []byte{0xaa,0xbb,0xcc,0xdd,0xee,0xff},},
	})
	if err != nil {
		t.Fatalf("failed insert data: %v", err)
	}

	// select data using QuerySQL
	r, err := c.QuerySQL(ignite3.QuerySQLData{
		Query:    "SELECT * FROM TEST_TYPES ORDER BY SINT ASC",
		PageSize: 10000,
	})
	if err != nil {
		t.Fatalf("failed query data: %v", err)
	}
	for _, row := range r.Rows {
		fmt.Printf("%v\n", row)
	}
}
