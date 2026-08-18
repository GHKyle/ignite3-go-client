package ignite3

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	//"time"

	"github.com/GHKyle/ignite3-go-client/binary/errors"
	"github.com/GHKyle/ignite3-go-client/debug"
)

const (
	MIN_HEARTBEAT_INTERVAL_MS = 500
)

// ConnInfo contains connections parameters
type ConnInfo struct {
	Network, Host       string
	Port                int
	Major, Minor, Patch int
	Username, Password  string
	Dialer              net.Dialer
	TLSConfig           *tls.Config
}

// Client is interface to communicate with Apache Ignite cluster.
// Client is thread safe.
type Client interface {
	// Connected return true if connection to the cluster is active
	Connected() bool

	// Do sends request and receives response
	Do(req Request, res Response) error

	// Close closes connection.
	// Returns:
	// nil in case of success.
	// error object in case of error.
	Close() error

	SendHeartbeat() error

	GetTables() (map[int64]string, error)

	GetTable(table string) error

	// Cache Configuration methods
	// See for details:
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations

	// CacheCreateWithName Creates a cache with a given name.
	// Cache template can be applied if there is a '*' in the cache name.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_create_with_name
	CacheCreateWithName(cache string) error

	// CacheGetOrCreateWithName creates a cache with a given name.
	// Cache template can be applied if there is a '*' in the cache name.
	// Does nothing if the cache exists.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_get_or_create_with_name
	CacheGetOrCreateWithName(cache string) error

	// CacheGetNames returns existing cache names.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_get_names
	CacheGetNames() ([]string, error)

	// CacheGetConfiguration gets configuration for the given cache.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_get_configuration
	/*CacheGetConfiguration(cache string, flag byte) (*CacheConfiguration, error)

	// CacheCreateWithConfiguration creates cache with provided configuration.
	// An error is returned if the name is already in use.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_create_with_configuration
	CacheCreateWithConfiguration(cc *CacheConfigurationRefs) error

	// CacheGetOrCreateWithConfiguration creates cache with provided configuration.
	// Does nothing if the name is already in use.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_get_or_create_with_configuration
	CacheGetOrCreateWithConfiguration(cc *CacheConfigurationRefs) error*/

	// CacheDestroy destroys cache with a given name.
	// https://apacheignite.readme.io/docs/binary-client-protocol-cache-configuration-operations#section-op_cache_destroy
	CacheDestroy(cache string) error

	// Key-Value Queries
	// See for details:
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations

	// CacheGet retrieves a value from cache by key.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get
	CacheGet(cache string, binary bool, key interface{}) (interface{}, error)

	// CacheGetAll retrieves multiple key-value pairs from cache.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_all
	CacheGetAll(cache string, binary bool, keys []interface{}) (map[interface{}]interface{}, error)

	// CachePut puts a value with a given key to cache (overwriting existing value if any).
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_put
	CachePut(cache string, binary bool, key interface{}, value interface{}) error

	// CachePutAll puts a value with a given key to cache (overwriting existing value if any).
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_put_all
	CachePutAll(cache string, binary bool, data map[interface{}]interface{}) error

	// CacheContainsKey returns a value indicating whether given key is present in cache.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_contains_key
	CacheContainsKey(cache string, binary bool, key interface{}) (bool, error)

	// CacheContainsKeys returns a value indicating whether all given keys are present in cache.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_contains_keys
	CacheContainsKeys(cache string, binary bool, keys []interface{}) (bool, error)

	// CacheGetAndPut puts a value with a given key to cache, and returns the previous value for that key.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_and_put
	CacheGetAndPut(cache string, binary bool, key interface{}, value interface{}) (interface{}, error)

	// CacheGetAndReplace puts a value with a given key to cache, returning previous value for that key,
	// if and only if there is a value currently mapped for that key.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_and_replace
	CacheGetAndReplace(cache string, binary bool, key interface{}, value interface{}) (interface{}, error)

	// CacheGetAndRemove removes the cache entry with specified key, returning the value.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_and_remove
	CacheGetAndRemove(cache string, binary bool, key interface{}) (interface{}, error)

	// CachePutIfAbsent puts a value with a given key to cache only if the key does not already exist.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_put_if_absent
	CachePutIfAbsent(cache string, binary bool, key interface{}, value interface{}) (bool, error)

	// CacheGetAndPutIfAbsent puts a value with a given key to cache only if the key does not already exist.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_and_put_if_absent
	CacheGetAndPutIfAbsent(cache string, binary bool, key interface{}, value interface{}) (interface{}, error)

	// CacheReplace puts a value with a given key to cache only if the key already exists.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_replace
	CacheReplace(cache string, binary bool, key interface{}, value interface{}) (bool, error)

	// CacheReplaceIfEquals puts a value with a given key to cache only if
	// the key already exists and value equals provided value.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_replace_if_equals
	CacheReplaceIfEquals(cache string, binary bool, key interface{}, valueCompare interface{}, valueNew interface{}) (bool, error)

	// CacheClear clears the cache without notifying listeners or cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_clear
	CacheClear(cache string, binary bool) error

	// CacheClearKey clears the cache key without notifying listeners or cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_clear_key
	CacheClearKey(cache string, binary bool, key interface{}) error

	// CacheClearKeys clears the cache keys without notifying listeners or cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_clear_keys
	CacheClearKeys(cache string, binary bool, keys []interface{}) error

	// CacheRemoveKey removes an entry with a given key, notifying listeners and cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_remove_key
	CacheRemoveKey(cache string, binary bool, key interface{}) (bool, error)

	// CacheRemoveIfEquals removes an entry with a given key if provided value is equal to actual value,
	// notifying listeners and cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_remove_if_equals
	CacheRemoveIfEquals(cache string, binary bool, key interface{}, value interface{}) (bool, error)

	// CacheGetSize gets the number of entries in cache.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_get_size
	CacheGetSize(cache string, binary bool, modes []byte) (int64, error)

	// CacheRemoveKeys removes entries with given keys, notifying listeners and cache writers.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_remove_keys
	CacheRemoveKeys(cache string, binary bool, keys []interface{}) error

	// CacheRemoveAll destroys cache with a given name.
	// https://apacheignite.readme.io/docs/binary-client-protocol-key-value-operations#section-op_cache_remove_all
	CacheRemoveAll(cache string, binary bool) error

	// SQL and Scan Queries
	// See for details:
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations

	// QuerySQL executes an SQL query over data stored in the cluster. The query returns the whole record (key and value).
	QuerySQL(data QuerySQLData) (QuerySQLResult, error)

	// QuerySQLCursorGetPage retrieves the next SQL query cursor page by cursor id from QuerySQL.
	// It needs the schema (we find it embedded in the metadata of original query)
	QuerySQLCursorGetPage(id int64, metadata *ResultSetMetadata) (QuerySQLResult, error)

	// QuerySQLFieldsRaw is equal to QuerySQLFields but return raw Response object.
	// Used for SQL driver to reduce memory allocations.
	QuerySQLFieldsRaw(data QuerySQLFieldsData) (*ResponseOperation, error)

	// QuerySQLFields performs SQL fields query.
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations#section-op_query_sql_fields
	QuerySQLFields(data QuerySQLFieldsData) (QuerySQLResult, error)

	// QuerySQLFieldsCursorGetPageRaw is equal to QuerySQLFieldsCursorGetPage but return raw Response object.
	// Used for SQL driver to reduce memory allocations.
	QuerySQLFieldsCursorGetPageRaw(id int64) (*ResponseOperation, error)

	// QuerySQLFieldsCursorGetPage retrieves the next query result page by cursor id from QuerySQLFields.
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations#section-op_query_sql_fields_cursor_get_page
	//QuerySQLFieldsCursorGetPage(id int64, fieldCount int) (QuerySQLFieldsPage, error)

	// QueryScan performs scan query.
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations#section-op_query_scan
	//QueryScan(cache string, binary bool, data QueryScanData) (QueryScanResult, error)

	// QueryScanCursorGetPage fetches the next SQL query cursor page by cursor id that is obtained from OP_QUERY_SCAN.
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations#section-op_query_scan_cursor_get_page
	//QueryScanCursorGetPage(id int64) (QueryScanPage, error)

	// ResourceClose closes a resource, such as query cursor.
	// https://apacheignite.readme.io/docs/binary-client-protocol-sql-operations#section-op_resource_close
	ResourceClose(id int64) error
}

type client struct {
	debugID   string
	conn      net.Conn
	mutex     *sync.Mutex
	quit      chan struct{}
	RequestId int64          // A request ID, incremented after each request (heartbeat, operation, ...) FIXME: Increment it!

	Client
}

// IsConnected return true if connection to the cluster is active
func (c *client) Connected() bool {
	return c.conn != nil
}

// Do sends request and receives response
func (c *client) Do(req Request, res Response) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// send request
	if _, err := req.WriteTo(c.conn); err != nil {
		return errors.Wrapf(err, "failed to send request to server")
	}
	// Increment client.RequestId
	c.RequestId++

	// receive response
	_, err := res.ReadFrom(bufio.NewReader(c.conn))

	return err
}

// Close closes connection.
// Returns:
// nil in case of success.
// error object in case of error.
func (c *client) Close() error {
	if c.Connected() {
		defer func() { c.conn = nil }()
		// Stop heartbeat task
		//close(c.quit)
		return c.conn.Close()
	}
	return nil
}

// Connect connects to the Apache Ignite cluster
// Returns: client
func Connect(ci ConnInfo) (Client, error) {
	address := fmt.Sprintf("%s:%d", ci.Host, ci.Port)

	// connect
	var conn net.Conn
	var err error
	if ci.TLSConfig != nil {
		conn, err = tls.DialWithDialer(&ci.Dialer, ci.Network, address, ci.TLSConfig)
	} else {
		conn, err = ci.Dialer.Dial(ci.Network, address)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open connection")
	}

	c := &client{conn: conn, debugID: strings.Join([]string{"network=", ci.Network, "', address='", address, "'"}, ""),
		mutex: &sync.Mutex{}}
	runtime.SetFinalizer(c, clientFinalizer)

	// request and response
	req := NewRequestHandshake(ci.Major, ci.Minor, ci.Patch, ci.Username, ci.Password)
	res := &ResponseHandshake{}

	// make handshake
	if err = c.Do(req, res); err != nil {
		c.Close()
		return nil, errors.Wrapf(err, "failed to make handshake")
	}

	if !res.Success {
		c.Close()
		return nil, errors.Errorf("handshake failed: %s, server supported protocol version is v%d.%d.%d",
			res.ErrorMessage, res.Major, res.Minor, res.Patch)
	}

	c.RequestId = 0
	// TODO : Adjust handshake interval by server config ?
	// FIXME: Crash when hearbeat sent while running query
	/*ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for {
			select {
				case <- ticker.C:
					c.SendHeartbeat()
				case <- c.quit:
					ticker.Stop()
					return
			}
		}
	}()*/
	

	// return connected client
	return c, nil
}


func (c *client) SendHeartbeat() error {
	req := NewRequestHeartbeat(c.RequestId)
	res := NewResponseHeartbeat(c.RequestId)
	c.RequestId++
	// execute operation
	if err := c.Do(req, res); err != nil {
		return errors.Wrapf(err, "failed to execute OP_HEARTBEAT operation")
	}
	if err := res.CheckStatus(); err != nil {
		return err
	}
	return nil
}

// clientFinalizer is resource leak spy
func clientFinalizer(c *client) {
	if c.Connected() {
		debug.ResourceLeakLogger.Printf("client \"%s\" is not closed", c.debugID)
		c.Close()
	}
}
