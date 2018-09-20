package main

import (
	"context"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/stats"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/tag"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"html/template"
	"log"
	"net/http"
	"time"

	"database/sql"
	"fmt"
	_ "github.com/SAP/go-hdb/driver"
	"github.com/opencensus-integrations/ocsql"
)

const (
	html = `<html>
	<head><title>opencensus example</title></head>
	<body>
	<h1>OpenCensus Example</h1>
	<p><a href="/result">result</a></p>
	<p><a href="/metrics">metrics</a></p>
	<p><a href="/debug/rpcz">rpcz</a></p>
	<p><a href="/debug/tracez">tracez</a></p>
	</body>
	</html>`
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("foo").Parse(html)
	if err != nil {
		log.Fatalf("Cannot parse template: %v", err)
	}
	t.Execute(w, "")
}

func main() {
	//	ctx := context.Background()

	// create prometheus exporter
	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "myservice",
	})

	if err != nil {
		log.Fatalf("Failed to create the Prometheus exporter: %v", err)
	}

	view.RegisterExporter(prometheusExporter)
	if err := view.Register(ochttp.DefaultClientViews...); err != nil {
		log.Fatalf("Failed to register http default client views for metrics: %v", err)
	}

	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.Fatalf("Failed to register http default server views for metrics: %v", err)
	}




	

	// create jaeger exporter

	agentEndpointURI := "10.58.137.243:6831"
	collectorEndpointURI := "http://10.58.137.243:14268"

	jaegerExporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpointURI,
		Endpoint:      collectorEndpointURI,
		ServiceName:   "myservice",
	})
	if err != nil {
		log.Fatalf("Failed to create the Jaeger exporter: %v", err)
	}

	
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	trace.RegisterExporter(jaegerExporter)
	// connect hana db
	driverName, err := ocsql.Register("hdb", ocsql.WithAllTraceOptions())

	if err != nil {
		log.Fatalf("Failed to register the ocsql driver: %v", err)
	}


	db, err := sql.Open(driverName, "hdb://SYSTEM:Toor1234@10.130.192.30:30015")

	if err != nil {
		log.Fatalf("Failed to open the HANA database: %v", err)
	}

	defer func() {
		db.Close()
		// Wait to 4 seconds so that the traces can be exported
		waitTime := 4 * time.Second
		log.Printf("Waiting for %s seconds to ensure all traces are exported before exiting", waitTime)
		<-time.After(waitTime)
	}()

	ctx, span := trace.StartSpan(context.Background(), "hana")
	defer span.End()

	fCtx, fSpan := trace.StartSpan(ctx, "get_value")
	
	row := db.QueryRowContext(fCtx, "select 1 from dummy")
	
	fSpan.End()

	var number string

	if err := row.Scan(&number); err != nil {
		log.Fatalf("Failed to get data: %v", err)
	}
	
	svcnamekey, err := tag.NewKey("myservice/name")
	if err != nil {
	 log.Fatal(err)
	}

	pectx, err := tag.New(ctx,
    tag.Insert(svcnamekey, "myservice"),
	)

	 if err != nil {
		log.Fatal(err)
	 }



	videoSize := stats.Int64("myservice/video_size", "processed video size", "MB")

	
	
	svcview:= view.View{
		Name: "my.org/video_size_distribution",
		Description: "distribution of processed video size over time",
		TagKeys: []tag.Key{svcnamekey},
		Measure: videoSize,
		Aggregation: view.LastValue(),
	}

	if err := view.Register(&svcview); err != nil {
	log.Fatal(err)
	}
	stats.Record(pectx, videoSize.M(102478))
	view.SetReportingPeriod(1 * time.Second)
	
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheusExporter)
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, html)
	})
	mux.Handle("/debug/", http.StripPrefix("/debug", zpages.Handler))
	h := &ochttp.Handler{Handler: mux}
	if err := http.ListenAndServe(":9999", h); err != nil {
		log.Fatalf("HTTP server ListenAndServe error: %v", err)
	}

}
