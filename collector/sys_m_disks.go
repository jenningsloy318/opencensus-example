// Scrape `sys_m_disks`.
package collector

import (
	"context"
	"database/sql"
	_ "github.com/SAP/go-hdb/driver"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"log"
)

const (
	// Scrape query.
	disksQuery = `SELECT HOST,PATH,USAGE_TYPE,TOTAL_SIZE,USED_SIZE FROM SYS.M_DISKS`
	// Subsystem.
	disks = "sys_m_disks"
)

var (
	hostTag, _         = tag.NewKey("host")
	pathTag, _         = tag.NewKey("path")
	usageTypeTag, _    = tag.NewKey("UsageType")
	total_size_measure = stats.Int64("total_size", "Volume Size.", "MB")
	used_size_measure  = stats.Int64("used_size", "Volume Used Space.", "MB")

	totalSizeView      = &view.View{
		Name:        "sys_m_disks/total_size",
		Description: "Volume Size.",
		TagKeys:     []tag.Key{hostTag, pathTag, usageTypeTag},
		Measure:     total_size_measure,
		Aggregation: view.LastValue(),
	}
	usedSizeView = &view.View{
		Name:        "sys_m_disks/used_size",
		Description: "Volume Size.",
		TagKeys:     []tag.Key{hostTag, pathTag, usageTypeTag},
		Measure:     used_size_measure,
		Aggregation: view.LastValue(),
	}

	SYS_M_DisksViews = []*view.View{
		totalSizeView,
		usedSizeView,
	}
)

func CreateView(ctx context.Context, db *sql.DB) {

	ctx, span := trace.StartSpan(ctx, "sql_query_sys_m_disks")
	//get the sql data
	defer span.End()

// if muliple row returned
disksRows ,err := db.QueryContext(ctx, disksQuery)
defer disksRows.Close()
	
if err !=nil {
	log.Fatalf("Failed to excute query: %v", err)

}
	var host string
	var path string
	var usage_type string
	var total_size int64
	var used_size int64

	sqlRowsScanCtx, sqlRowsScanSpan := trace.StartSpan(ctx, "sql_rows_scan")

	for disksRows.Next() {
		if err := disksRows.Scan(&host, &path, &usage_type, &total_size, &used_size); err != nil {
			log.Fatalf("Failed to featch rows: %v", err)
		}
	}

	defer sqlRowsScanSpan.End()


	measureSetCtx, measureSetCtxSpan := trace.StartSpan(sqlRowsScanCtx, "measure_value_set")
	diskctx, err := tag.New(measureSetCtx,
		tag.Insert(hostTag, host),
		tag.Insert(pathTag, path),
		tag.Insert(usageTypeTag, usage_type),
	)

	if err !=nil {
		log.Fatalf("Failed to insert tag: %v", err)

	}


	stats.Record(diskctx, total_size_measure.M(total_size))
	stats.Record(diskctx, used_size_measure.M(used_size))

	measureSetCtxSpan.End()
}