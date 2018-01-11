package complexdriver

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/url"
	"time"

	cache "github.com/querian/aawql/csvcache"
	"github.com/querian/aawql/db"
	awql "github.com/querian/aawql/driver"
)

// AdvancedDriver implements all methods to pretend as a sql database driver.
// It is an advanced version of Awql driver.
// It adds cache, the possibility to get database details.
type AdvancedDriver struct{}

// init adds advanced awql as sql database driver.
// @see https://github.com/golang/go/wiki/SQLDrivers
func init() {
	sql.Register("aawql", &AdvancedDriver{})
}

// Open returns a new connection to the database.
// @see DatabaseDir:CacheDir:WithCache|AdwordsId[:ApiVersion:SupportsZeroImpressions]|DeveloperToken[|ClientId][|ClientSecret][|RefreshToken]
// @example /data/base/dir:/cache/dir:false|123-456-7890:v201607:true|dEve1op3er7okeN|1234567890-c1i3n7iD.com|c1ien753cr37|1/R3Fr35h-70k3n
func (d *AdvancedDriver) Open(dsn string) (conn driver.Conn, err error) {

	var val url.Values
	val, err = awql.ParseDSN(dsn)
	if err != nil {
		return
	}

	if val.Get("skip_column_header") != "1" {
		val.Set("skip_column_header", "1")
	}

	// init underlying driver
	// Wraps the Awql driver.
	dd := &awql.Driver{}
	conn, err = dd.Open("adwords?" + val.Encode())
	if err != nil {
		return
	}

	var useCache bool
	var cacheDir string
	var cacheDuration time.Duration
	if cacheDir = val.Get("cache_dir"); cacheDir != "" && val.Get("cache") == "1" {
		useCache = true
		if cd := val.Get("cache_duration"); cd != "" {
			if cacheDuration, err = time.ParseDuration(cd); err != nil {
				err = errors.New("invalid duration")
				return
			}
		}
	}

	var c *cache.Cache
	if useCache {
		// Initializes the cache to save result sets inside.
		if cacheDuration == 0 {
			cacheDuration = 24 * time.Hour
		}
		c = cache.New(cacheDir, cacheDuration)
		// TODO make file/goroutine safe
		c.FlushAll()
	}

	// path where the user definition are stored
	databaseDir := val.Get("database_dir")

	// Loads all information about the database.
	awqlDb, err := db.Open(val.Get("version") + "|" + databaseDir)
	if err != nil {
		return nil, err
	}
	return &Conn{cn: conn.(*awql.Conn), fc: c, db: awqlDb, id: val.Get("adwords_id")}, nil
}

// Conn represents a connection to a database and implements driver.Conn.
type Conn struct {
	cn *awql.Conn
	db *db.Database
	fc *cache.Cache
	id string
}

// Close marks this connection as no longer in use.
func (c *Conn) Close() error {
	return c.cn.Close()
}

// Begin is dedicated to start a transaction and awql does not support it.
func (c *Conn) Begin() (driver.Tx, error) {
	return c.cn.Begin()
}

// Prepare returns a prepared statement, bound to this connection.
func (c *Conn) Prepare(q string) (driver.Stmt, error) {
	if q == "" {
		// No query to prepare.
		return nil, io.EOF
	}
	return &Stmt{si: &awql.Stmt{Db: c.cn, SrcQuery: q}, db: c.db, fc: c.fc, id: c.id}, nil
}

// Result is the result of a query execution.
type Result struct {
	err error
}

// LastInsertId returns the database's auto-generated ID
// after, for example, an INSERT into a table with primary key.
func (r *Result) LastInsertId() (int64, error) {
	return 0, driver.ErrSkip
}

// RowsAffected returns the number of rows affected by the query.
func (r *Result) RowsAffected() (int64, error) {
	return 0, r.err
}
