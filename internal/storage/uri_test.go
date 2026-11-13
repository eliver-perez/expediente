package storage

import (
	"net/url"
	"strings"
	"testing"
)

func TestPostgresCredentialsAreURLSafe(t *testing.T) {
	dsn := postgresDSN(55432, "aidex_app", "a?#:@/ password", "aidex")
	address, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := address.User.Password()
	if password != "a?#:@/ password" || address.Host != "127.0.0.1:55432" {
		t.Fatal("credentials or host changed")
	}
}
func TestParameterBindingPreservesLiteralsAndBodies(t *testing.T) {
	query := "SELECT ?, '?' AS literal, \"?\" AS identifier, $$ ? $$, $fn$ ? $fn$ -- ?\n/* ? */ WHERE x=?"
	got := bindParameters(query)
	if !strings.HasPrefix(got, "SELECT $1,") || !strings.HasSuffix(got, "WHERE x=$2") || strings.Count(got, "$1") != 1 {
		t.Fatal(got)
	}
}
