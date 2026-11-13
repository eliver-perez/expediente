//go:build development

package libraries

import (
	"context"
	"testing"
	"time"
)

func TestProgressAndDashboardWithGeneralReadersExhausted(t *testing.T) {
	service, principal, _ := fixture(t)
	library := addLibrary(t, service, principal, "Concurrent reads")
	connections := make([]interface{ Close() error }, 0, service.Database.Reader.Stats().MaxOpenConnections)
	for i := 0; i < service.Database.Reader.Stats().MaxOpenConnections; i++ {
		connection, err := service.Database.Reader.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, connection)
	}
	defer func() {
		for _, connection := range connections {
			connection.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started := time.Now()
	report, err := service.Processing(ctx, principal, library)
	if err != nil {
		t.Fatal(err)
	}
	if report.StatsAvailable || report.PerformanceAvailable {
		t.Fatalf("busy general readers should only suppress optional metrics: %+v", report)
	}
	if elapsed := time.Since(started); elapsed > 2500*time.Millisecond {
		t.Fatalf("progress took %s", elapsed)
	}
	if _, err := service.Dashboard(ctx, principal); err != nil {
		t.Fatal(err)
	}
}
