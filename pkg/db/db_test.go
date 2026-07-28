package db

import "testing"

func TestMySQLOptionsDataSourceNameUsesExplicitDSN(t *testing.T) {
	want := "ops:secret@tcp(db.example.com:3307)/miniblog?charset=utf8mb4&parseTime=true&loc=Local"
	opts := &MySQLOptions{
		DSN:      want,
		Host:     "ignored-host",
		Port:     "3306",
		Username: "ignored-user",
		Password: "ignored-password",
		Database: "ignored-database",
	}

	if got := opts.DataSourceName(); got != want {
		t.Fatalf("DataSourceName() = %q, want %q", got, want)
	}
	if got := opts.DNS(); got != want {
		t.Fatalf("DNS() compatibility result = %q, want %q", got, want)
	}
}

func TestMySQLOptionsDataSourceNameBuildsLegacyFields(t *testing.T) {
	opts := &MySQLOptions{
		Host:     "db.example.com",
		Port:     "3307",
		Username: "ops",
		Password: "secret",
		Database: "miniblog",
	}
	want := "ops:secret@tcp(db.example.com:3307)/miniblog?charset=utf8mb4&parseTime=true&loc=Local"

	if got := opts.DataSourceName(); got != want {
		t.Fatalf("DataSourceName() = %q, want %q", got, want)
	}
}
