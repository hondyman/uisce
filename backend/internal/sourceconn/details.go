package sourceconn

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
)

// ErrBadConfig: the datasource's connection details cannot describe a connection.
var ErrBadConfig = errors.New("sourceconn: invalid connection details")

// details is the shape of a datasource's connection config: the nested form the database stores
// (auth.basic) and the flat form the connection form sends, plus a pass-through DSN and mTLS client
// certificate auth.
type details struct {
	Auth struct {
		Basic struct {
			Password string `json:"password"`
			Username string `json:"username,omitempty"`
		} `json:"basic"`
	} `json:"auth"`
	Database string `json:"database"`
	Host     string `json:"host"`
	Password string `json:"password,omitempty"`
	Port     int    `json:"port"`
	Schema   string `json:"schema,omitempty"`
	SSL      bool   `json:"ssl,omitempty"`
	SSLMode  string `json:"sslMode,omitempty"`
	Username string `json:"username,omitempty"`
	Type     string `json:"type"`
	DSN      string `json:"dsn,omitempty"`

	// mTLS client-certificate auth ("Key Pair" in the connection form). The values are PEM content,
	// not paths: the private key is parsed into an in-memory tls.Config and never touches disk.
	AuthType   string `json:"auth_type,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	CACert     string `json:"ca_cert,omitempty"`
}

// connConfig turns hydrated connection details into a pgx connection config. It is the one place
// that interprets a datasource's connection config (it replaces the copy that lived in
// metadata.connectToDatabaseFromDetails). warnings carries anything the caller should log, such as
// an mTLS config that does not verify the server.
func connConfig(raw []byte) (cfg *pgx.ConnConfig, host, database, sslMode string, warnings []string, err error) {
	var d details
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, "", "", "", nil, fmt.Errorf("%w: %v", ErrBadConfig, err)
	}
	if d.Auth.Basic.Username != "" {
		d.Username = d.Auth.Basic.Username
	}
	if d.Auth.Basic.Password != "" {
		d.Password = d.Auth.Basic.Password
	}
	if d.SSLMode == "" {
		if d.SSL {
			d.SSLMode = "require"
		} else {
			d.SSLMode = "disable"
		}
	}

	dsn := d.DSN
	if dsn == "" {
		switch {
		case d.Host == "":
			return nil, "", "", "", nil, fmt.Errorf("%w: host is empty", ErrBadConfig)
		case d.Port <= 0 || d.Port > 65535:
			return nil, "", "", "", nil, fmt.Errorf("%w: port must be between 1 and 65535 (got %d)", ErrBadConfig, d.Port)
		case d.Username == "":
			return nil, "", "", "", nil, fmt.Errorf("%w: missing username", ErrBadConfig)
		case d.Database == "":
			return nil, "", "", "", nil, fmt.Errorf("%w: missing database name", ErrBadConfig)
		}
		u := url.URL{
			Scheme: "postgres",
			Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
			Path:   d.Database,
			User:   url.UserPassword(d.Username, d.Password),
		}
		q := u.Query()
		q.Set("sslmode", d.SSLMode)
		if d.Schema != "" {
			q.Set("search_path", d.Schema)
		}
		u.RawQuery = q.Encode()
		dsn = u.String()
	}

	cc, perr := pgx.ParseConfig(dsn)
	if perr != nil {
		return nil, "", "", "", nil, fmt.Errorf("%w: %v", ErrBadConfig, perr)
	}

	if d.AuthType == "key_pair" && d.ClientCert != "" && d.PrivateKey != "" {
		cert, cerr := tls.X509KeyPair([]byte(d.ClientCert), []byte(d.PrivateKey))
		if cerr != nil {
			return nil, "", "", "", nil, fmt.Errorf("%w: client certificate/key for key_pair auth: %v", ErrBadConfig, cerr)
		}
		tc := &tls.Config{Certificates: []tls.Certificate{cert}, ServerName: d.Host, MinVersion: tls.VersionTLS12}
		if d.CACert != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(d.CACert)) {
				return nil, "", "", "", nil, fmt.Errorf("%w: ca_cert for key_pair auth has no certificates", ErrBadConfig)
			}
			tc.RootCAs = pool
		} else {
			// No CA to verify the server against: client-authenticated, but the server's identity
			// is not verified. Acceptable for local/dev, not for production.
			tc.InsecureSkipVerify = true
			warnings = append(warnings, fmt.Sprintf("key_pair auth for %s has no ca_cert; the server certificate will not be verified", d.Host))
		}
		cc.TLSConfig = tc
	}

	host = cc.Host
	database = cc.Database
	return cc, host, database, d.SSLMode, warnings, nil
}
