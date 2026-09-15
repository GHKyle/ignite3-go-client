package ignite3_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	ignite3 "github.com/GHKyle/ignite3-go-client/binary/v1"
	_ "github.com/GHKyle/ignite3-go-client/sql"
)

// TestTableKvLive is an end-to-end check of the binary tuple (key-value) write path against a
// live cluster. It is skipped unless IGNITE_TEST_HOST is set, for example:
//
//	IGNITE_TEST_HOST=127.0.0.1:10800 go test -run TestTableKvLive -v ./binary/v1/
//
// The test creates its own schema and table and drops them at the end, so it does not touch
// user data. It verifies the two properties that CDC sinks depend on:
//   - writing a row twice with the same key overwrites it (upsert semantics), and
//   - NULL columns survive the round trip.
func TestTableKvLive(t *testing.T) {
	host := os.Getenv("IGNITE_TEST_HOST")
	if host == "" {
		t.Skip("set IGNITE_TEST_HOST=host:port to run the live test")
	}

	idx := strings.LastIndex(host, ":")
	if idx < 0 {
		t.Fatalf("invalid IGNITE_TEST_HOST: %s", host)
	}
	hostName, portText := host[:idx], host[idx+1:]

	var port int
	if _, err := fmt.Sscanf(portText, "%d", &port); err != nil {
		t.Fatalf("invalid port in IGNITE_TEST_HOST: %s", host)
	}

	db, err := sql.Open("ignite3", fmt.Sprintf("tcp://%s/PUBLIC?schema=PUBLIC&version=3.0.0", host))
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("sql ping: %v", err)
	}

	const schema = `"ZZ_KV"`
	const table = `"ZZ_KV"."T"`

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = db.ExecContext(cleanupCtx, `DROP TABLE IF EXISTS `+table)
		_, _ = db.ExecContext(cleanupCtx, `DROP SCHEMA IF EXISTS `+schema)
	}()

	exec := func(label, query string) {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}

	exec("create schema", `CREATE SCHEMA IF NOT EXISTS `+schema)
	exec("create table", `CREATE TABLE IF NOT EXISTS `+table+
		` ("ID" VARCHAR(36) PRIMARY KEY, "V" VARCHAR(50), "N" INT, "TS" TIMESTAMP, "D" DECIMAL(20,2))`)

	cli, err := ignite3.Connect(ignite3.ConnInfo{Network: "tcp", Host: hostName, Port: port, Major: 3, Minor: 0, Patch: 0})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cli.Close()

	tbl, err := cli.GetTableByName(table)
	if err != nil {
		if tables, terr := cli.GetTables(); terr == nil {
			for id, name := range tables {
				t.Logf("  known table %d => %q", id, name)
			}
		}
		t.Fatalf("GetTableByName: %v", err)
	}
	t.Logf("table id=%d name=%s schemaVersion=%d", tbl.Id, tbl.Name, tbl.Schema.Version)
	for _, c := range tbl.Schema.Columns {
		t.Logf("  column %-6s typeId=%-3d kind=%-2d key=%-5v scale=%d", c.Name, c.TypeId, c.Kind, c.Key, c.Scale)
	}

	ts := ignite3.Timestamp{Time: time.Date(2026, 4, 20, 16, 29, 40, 644838000, time.UTC)}
	dec := ignite3.Decimal{Unscaled: big.NewInt(12345), Scale: 2}

	// 1) Two rows, the second one with NULL columns.
	if err := tbl.UpsertAll([]map[string]interface{}{
		{"ID": "k1", "V": "v1", "N": int32(11), "TS": ts, "D": dec},
		{"ID": "k2", "V": "v2", "N": nil, "TS": nil, "D": nil},
	}); err != nil {
		t.Fatalf("UpsertAll #1: %v", err)
	}
	checkKvState(t, ctx, db, table, 2, "k1", "v1")

	// 2) Same key again: the row must be OVERWRITTEN (this is the CDC "update" case).
	if err := tbl.UpsertAll([]map[string]interface{}{
		{"ID": "k1", "V": "v1-updated", "N": int32(22), "TS": ts, "D": dec},
	}); err != nil {
		t.Fatalf("UpsertAll #2: %v", err)
	}
	checkKvState(t, ctx, db, table, 2, "k1", "v1-updated")

	// 3) Delete by key.
	if err := tbl.DeleteAll([]map[string]interface{}{{"ID": "k2"}}); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	checkKvState(t, ctx, db, table, 1, "k1", "v1-updated")

	// 4) The NULL written for k2 must be readable as NULL before the delete; re-insert and check.
	if err := tbl.UpsertAll([]map[string]interface{}{
		{"ID": "k3", "V": nil, "N": nil, "TS": nil, "D": nil},
	}); err != nil {
		t.Fatalf("UpsertAll #3: %v", err)
	}
	var nullable sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT "V" FROM `+table+` WHERE "ID" = ?`, "k3").Scan(&nullable); err != nil {
		t.Fatalf("select nullable: %v", err)
	}
	if nullable.Valid {
		t.Fatalf("k3.V = %q, want NULL", nullable.String)
	}
}

// checkKvState verifies the row count of the table and the value stored under the given key.
func checkKvState(t *testing.T, ctx context.Context, db *sql.DB, table string, wantRows int, key, wantValue string) {
	t.Helper()

	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != wantRows {
		t.Fatalf("row count = %d, want %d", rows, wantRows)
	}

	var value sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT "V" FROM `+table+` WHERE "ID" = ?`, key).Scan(&value); err != nil {
		t.Fatalf("select %s: %v", key, err)
	}
	if value.String != wantValue {
		t.Fatalf("value of %s = %q, want %q", key, value.String, wantValue)
	}
}
