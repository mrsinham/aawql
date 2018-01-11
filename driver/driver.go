package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	awql "github.com/querian/aawql/net/oauth2"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Data source name.
const (
	APIVersion = "v201705"
	DsnSep     = "|"
	DsnOptSep  = ":"
)

var (
	httpClients     map[string]*http.Client
	httpClientsLock sync.Mutex
)

func init() {
	httpClients = make(map[string]*http.Client)
}

// Driver implements all methods to pretend as a sql database driver.
type Driver struct {
	client *http.Client
}

func RegisterHTTPClient(name string, c *http.Client) {
	httpClientsLock.Lock()
	httpClients[name] = c
	httpClientsLock.Unlock()
}

func UnregisterHTTPClient(name string) {
	httpClientsLock.Lock()
	delete(httpClients, name)
	httpClientsLock.Unlock()
}

// init adds  awql as sql database driver
// @see https://github.com/golang/go/wiki/SQLDrivers
// @implements https://golang.org/src/database/sql/driver/driver.go
func init() {
	sql.Register("awql", &Driver{})
}

// Open returns a new connection to the database.
// @see https://github.com/rvflash/awql-driver#data-source-name for how
// the DSN string is formatted
func (d *Driver) Open(dsn string) (driver.Conn, error) {
	conn, err := unmarshal(dsn)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// parseDsn returns an pointer to an Conn by parsing a DSN string.
// It throws an error on fails to parse it.
// dsn example : adwords?access_token=xxx&developer_token=xxxx&timeout=3s&version=201705
// dsn example : adwords?http_client=twenga&version=201706 : read the http client into the map
// dsn example : adwords?refresh_token=xxxx&client_id=xxxx&client_secret=xxxx&developper_token=xxxx&version=201705
// possible param :
// * zero_impression=1
// * raw_enum=1
func unmarshal(dsn string) (conn *Conn, err error) {

	var val url.Values
	val, err = ParseDSN(dsn)
	if err != nil {
		return
	}

	var c *http.Client

	httpClient := val.Get("http_client")
	accessToken := val.Get("access_token")
	refreshToken := val.Get("refresh_token")
	developerToken := val.Get("developer_token")

	ctx := context.Background()
	if to := val.Get("timeout"); to != "" {
		var d time.Duration
		d, err = time.ParseDuration(to)
		if err != nil {
			return
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}

	switch {
	case httpClient != "":
		httpClientsLock.Lock()
		c = httpClients[httpClient]
		httpClientsLock.Unlock()
		if c == nil {
			err = fmt.Errorf("no http client with the name %q", httpClient)
			return
		}
	case accessToken != "":
		if developerToken == "" {
			err = ErrDevToken
			return
		}
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: accessToken,
		})
		c = awql.NewClient(ctx, ts, developerToken)
	case refreshToken != "":
		if developerToken == "" {
			err = ErrDevToken
			return
		}
		clientId := val.Get("client_id")
		clientSecret := val.Get("client_secret")
		conf := &oauth2.Config{
			ClientID:     clientId,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
		}
		c = awql.NewClient(ctx, conf.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}), developerToken)
		c.Timeout = apiTimeout
	}

	conn = &Conn{}
	conn.client = c

	var adwordsID string
	if adwordsID = val.Get("adwords_id"); adwordsID == "" {
		err = ErrAdwordsID
		return
	}

	conn.adwordsID = adwordsID

	var includeZeroImpressions bool
	if val.Get("zero_impression") == "1" {
		includeZeroImpressions = true
	}
	var skipColumnHeader bool
	if val.Get("skip_column_header") == "1" {
		skipColumnHeader = true
	}
	var rawEnumValues bool
	if val.Get("raw_enum") == "1" {
		rawEnumValues = true
	}
	var version string
	if version = val.Get("version"); version == "" {
		version = APIVersion
	}

	conn.opts = NewOpts(version, includeZeroImpressions, skipColumnHeader, rawEnumValues)

	return conn, err
}

// Opts lists the available Adwords API properties.
type Opts struct {
	Version string
	SkipReportHeader,
	SkipColumnHeader,
	SkipReportSummary,
	IncludeZeroImpressions,
	UseRawEnumValues bool
}

// NewOpts returns a Opts with default options.
func NewOpts(version string, includeZeroImpressions, skipColumnHeader, rawEnums bool) *Opts {
	if version == "" {
		version = APIVersion
	}
	return &Opts{
		IncludeZeroImpressions: includeZeroImpressions,
		SkipColumnHeader:       skipColumnHeader,
		SkipReportHeader:       true,
		SkipReportSummary:      true,
		UseRawEnumValues:       rawEnums,
		Version:                version,
	}
}
