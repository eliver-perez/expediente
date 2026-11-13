package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
)

// Positional parameters are numbered once at the PostgreSQL driver boundary.
// Services can compose parameterized fragments without renumbering parameters.
// This performs no SQL dialect conversion and never interpolates values.
func bindParameters(query string) string {
	var out strings.Builder
	quote := byte(0)
	parameter := 0
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if quote != 0 {
			out.WriteByte(ch)
			if ch == quote {
				if i+1 < len(query) && query[i+1] == quote {
					i++
					out.WriteByte(quote)
				} else {
					quote = 0
				}
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			out.WriteByte(ch)
			continue
		}
		if ch == '$' {
			end := i + 1
			for end < len(query) && (query[end] == '_' || query[end] >= 'a' && query[end] <= 'z' || query[end] >= 'A' && query[end] <= 'Z') {
				end++
			}
			if end < len(query) && query[end] == '$' {
				tag := query[i : end+1]
				if close := strings.Index(query[end+1:], tag); close >= 0 {
					stop := end + 1 + close + len(tag)
					out.WriteString(query[i:stop])
					i = stop - 1
					continue
				}
			}
		}
		if i+1 < len(query) && (ch == '-' && query[i+1] == '-' || ch == '/' && query[i+1] == '*') {
			end := len(query)
			if ch == '-' {
				if n := strings.IndexByte(query[i:], '\n'); n >= 0 {
					end = i + n + 1
				}
			} else if n := strings.Index(query[i+2:], "*/"); n >= 0 {
				end = i + 2 + n + 2
			}
			out.WriteString(query[i:end])
			i = end - 1
			continue
		}
		if ch == '?' {
			parameter++
			fmt.Fprintf(&out, "$%d", parameter)
		} else {
			out.WriteByte(ch)
		}
	}
	return out.String()
}

type postgresConnector struct{ driver.Connector }

func (c postgresConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &postgresConnection{conn.(*stdlib.Conn)}, nil
}

type postgresConnection struct{ *stdlib.Conn }

func (c *postgresConnection) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}
func (c *postgresConnection) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.Conn.PrepareContext(ctx, bindParameters(query))
}
func (c *postgresConnection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.Conn.ExecContext(ctx, bindParameters(query), args)
}
func (c *postgresConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.Conn.QueryContext(ctx, bindParameters(query), args)
}
func (c *postgresConnection) CheckNamedValue(value *driver.NamedValue) error {
	// pgx chooses the appropriate codec from the SQL parameter type. Existing
	// constrained 0/1 columns and native boolean predicates both accept Go flags.
	if v, ok := value.Value.(bool); ok {
		value.Value = postgresFlag(v)
	}
	return c.Conn.CheckNamedValue(value)
}

func openPostgresPool(dsn, schema string, readOnly bool, limit int) (*sql.DB, error) {
	configuration, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL connection configuration")
	}
	configuration.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize() + ",public"
	configuration.RuntimeParams["application_name"] = "AIDEX 1.0.0"
	configuration.RuntimeParams["timezone"] = "UTC"
	configuration.RuntimeParams["lock_timeout"] = "5000"
	configuration.RuntimeParams["idle_in_transaction_session_timeout"] = "60000"
	if readOnly {
		configuration.RuntimeParams["default_transaction_read_only"] = "on"
	}
	pool := sql.OpenDB(postgresConnector{stdlib.GetConnector(*configuration)})
	pool.SetMaxOpenConns(limit)
	pool.SetMaxIdleConns(limit)
	return pool, nil
}

type postgresFlag bool

func (v postgresFlag) Int64Value() (pgtype.Int8, error) {
	var n int64
	if v {
		n = 1
	}
	return pgtype.Int8{Int64: n, Valid: true}, nil
}
func (v postgresFlag) BoolValue() (pgtype.Bool, error) {
	return pgtype.Bool{Bool: bool(v), Valid: true}, nil
}
