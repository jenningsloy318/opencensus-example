package collector

import (
	"database/sql"
	_ "github.com/SAP/go-hdb/driver"
	"github.com/opencensus-integrations/ocsql"
	"go.opencensus.io/trace"
	//	"go.opencensus.io/exporter/prometheus"
	"context"
	"log"
	"time"
)

func NEW(ctx context.Context, dsn string) {

	sqlDriverRegCtx, sqlDriverRegSpan := trace.StartSpan(ctx, "sql_register_driver")

	driverName, err := ocsql.Register("hdb", ocsql.WithAllTraceOptions())

	if err != nil {
		log.Fatalf("Failed to register the ocsql driver: %v", err)
	}

	sqlDriverRegSpan.End()

	sqlOpenCtx, sqlOpenSpan := trace.StartSpan(sqlDriverRegCtx, "sql_open_db_conn")

	db, err := sql.Open(driverName, dsn)

	if err != nil {
		log.Fatalf("Failed to open the HANA database: %v", err)
	}
	sqlOpenSpan.End()

	defer func() {
		db.Close()
		// Wait to 1 seconds so that the traces can be exported
		waitTime := 1 * time.Second
			log.Printf("Waiting for %s seconds to ensure all traces are exported before exiting", waitTime)
		<-time.After(waitTime)
	}()

	sqlQueryctx, sqlQuerySpan := trace.StartSpan(sqlOpenCtx, "sql_queries")
	defer sqlQuerySpan.End()
	CreateView(sqlQueryctx, db)

}
