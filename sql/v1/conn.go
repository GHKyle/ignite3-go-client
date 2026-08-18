package v1

import (
	"context"
	"database/sql/driver"
	"fmt"
	"runtime"
	"sort"
	"time"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
	ignite3 "github.com/GHKyle/ignite3-go-client/binary/v1"
	"github.com/GHKyle/ignite3-go-client/debug"
	"github.com/GHKyle/ignite3-go-client/sql/common"
)

// SQL connection struct
type conn struct {
	debugID string
	info    common.ConnInfo
	client  ignite3.Client

	driver.Conn
	driver.ExecerContext
	driver.Pinger
	driver.QueryerContext
	driver.NamedValueChecker
}

// isConnected return true if connection to the cluster is active
func (c *conn) isConnected() bool {
	return c.client != nil && c.client.Connected()
}

// resourceClose closes a resource, such as query cursor.
func (c *conn) resourceClose(id int64) error {
	if !c.isConnected() {
		return driver.ErrBadConn
	}
	return c.client.ResourceClose(id)
}

// <driver.Conn>

// Prepare returns a prepared statement, bound to this connection.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	if !c.isConnected() {
		return nil, driver.ErrBadConn
	}
	return newStmt(c, query), nil
}

// Close invalidates and potentially stops any current
// prepared statements and transactions, marking this
// connection as no longer in use.
//
// Because the sql package maintains a free pool of
// connections and only calls Close when there's a surplus of
// idle connections, it shouldn't be necessary for drivers to
// do their own connection caching.
func (c *conn) Close() error {
	if c.client != nil {
		defer func() {
			c.client = nil
		}()
		return c.client.Close()
	}
	return nil
}

// Begin starts and returns a new transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *conn) Begin() (driver.Tx, error) {
	return nil, errors.Errorf("Transactions are currently not supported by Ignite3 driver")
}

// </driver.Conn>

// <driver.ExecerContext>

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !c.isConnected() {
		return nil, driver.ErrBadConn
	}

	d := ignite3.QuerySQLFieldsData{
		Schema:           c.info.Schema,
		PageSize:         10000,
		Query:            query,
		Timeout:          c.info.Timeout,
	}
	if args != nil {
		l := len(args)
		d.QueryArgs = make([]interface{}, 0, l)
		if l > 0 {
			// sort slice
			tmp := make([]driver.NamedValue, l)
			copy(tmp, args)
			sort.Slice(tmp, func(i, j int) bool {
				return tmp[i].Ordinal < tmp[j].Ordinal
			})
			for i := 0; i < l; i++ {
				d.QueryArgs = append(d.QueryArgs, tmp[i].Value)
			}
		}
	}

	res, err := c.client.QuerySQLFields(d)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute query")
	}

	if res.HasMore {
		if err = c.resourceClose(res.ResponseId); err != nil {
			return nil, errors.Wrapf(err, "failed to close server cursor")
		}
	}

	if res.AffectedRows > 0 {
		return newResult(res.AffectedRows)
	}

	if !res.HasRowSet {
		return driver.ResultNoRows, nil
	}

	if len(res.Rows) == 0 {
		return driver.ResultNoRows, nil
	}

	return driver.ResultNoRows, nil
}

// </driver.ExecerContext>

// <driver.Pinger>

func (c *conn) Ping(ctx context.Context) error {
	r, err := c.QueryContext(ctx, "SELECT 1", nil)
	if err != nil {
		return errors.Wrapf(err, "failed to execute ping query")
	}
	var dest [1]driver.Value
	if err = r.Next(dest[:]); err != nil {
		return errors.Wrapf(err, "failed to read ping query response")
	}
	if "1" != fmt.Sprintf("%v", dest[0]) {
		return errors.Wrapf(err, "ping query returned unexpected value: %v", dest[0])
	}
	return nil
}

// </driver.Pinger>

// <driver.QueryerContext>

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !c.isConnected() {
		return nil, driver.ErrBadConn
	}

	d := ignite3.QuerySQLFieldsData{
		Schema:            c.info.Schema,
		PageSize:          c.info.PageSize,
		Query:             query,
		Timeout:           c.info.Timeout,
	}

	if args != nil {
		l := len(args)
		d.QueryArgs = make([]interface{}, 0, l)
		if l > 0 {
			// sort slice
			tmp := make([]driver.NamedValue, l)
			copy(tmp, args)
			sort.Slice(tmp, func(i, j int) bool {
				return tmp[i].Ordinal < tmp[j].Ordinal
			})
			for i := 0; i < l; i++ {
				d.QueryArgs = append(d.QueryArgs, tmp[i].Value)
			}
		}
	}

	r, err := c.client.QuerySQLFieldsRaw(d)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute query")
	}

	return newRows(c, r)
}

// </driver.QueryerContext>

// <driver.NamedValueChecker>

func (c *conn) CheckNamedValue(val *driver.NamedValue) error {
	// Ignore and handle later
	return nil
}

// </driver.NamedValueChecker>

func (c *conn) QueryNexPageContext(ctx context.Context, cursorID int64) (*ignite3.ResponseOperation, error) {
	if !c.isConnected() {
		return nil, driver.ErrBadConn
	}
	return c.client.QuerySQLFieldsCursorGetPageRaw(cursorID)
}

// Connect opens connection with protocol version v1
func Connect(ci common.ConnInfo) (driver.Conn, error) {
	if ci.Timeout > 0 {
		ci.ConnInfo.Dialer.Timeout = time.Duration(ci.Timeout) * time.Millisecond
	}

	client, err := ignite3.Connect(ci.ConnInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	c := &conn{info: ci, client: client, debugID: ci.URL}
	runtime.SetFinalizer(c, connFinalizer)
	return c, nil
}

// connFinalizer is memory leak spy
func connFinalizer(c *conn) {
	if c.isConnected() {
		debug.ResourceLeakLogger.Printf("connection \"%s\" is not closed", c.debugID)
		c.Close()
	}
}
