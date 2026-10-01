//go:build integration

package mastering

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync/atomic"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// queryCounter counts the statements sent to the database, so tests can
// see the round trips a stage costs.
type queryCounter struct{ n atomic.Int64 }

func (c *queryCounter) open(dsn string) (*sqlx.DB, error) {
	base, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(sql.OpenDB(countingConnector{base, c}), "postgres"), nil
}

type countingConnector struct {
	base *pq.Connector
	c    *queryCounter
}

func (k countingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := k.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return countingConn{conn, k.c}, nil
}

func (k countingConnector) Driver() driver.Driver { return k.base.Driver() }

type countingConn struct {
	driver.Conn
	c *queryCounter
}

func (w countingConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	w.c.n.Add(1)
	return w.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}

func (w countingConn) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	w.c.n.Add(1)
	return w.Conn.(driver.ExecerContext).ExecContext(ctx, q, args)
}

func (w countingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return w.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
}
