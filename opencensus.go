package main

import (
	"context"
	"fmt"
	_ "github.com/SAP/go-hdb/driver"
	"github.com/jenningsloy318/opencensus-example/collector"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
	"html/template"
	"log"
	"net/http"
	"time"
)

const (
	html = `<html>
	<head><title>opencensus example</title></head>
	<body>
	<h1>OpenCensus Example</h1>
	<p><a href="/metrics">metrics</a></p>
	<p><a href="/debug/rpcz">rpcz</a></p>
	<p><a href="/debug/tracez">tracez</a></p>
	</body>
	</html>`
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("home").Parse(html)
	if err != nil {
		log.Fatalf("Cannot parse template: %v", err)
	}
	t.Execute(w, "")
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	// retrieve target from request

	target := r.URL.Query().Get("target")

	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}
	log.Printf("Scraping target '%s'", target)

	// get the user and password from config
	user := ""
	password := ""

	// construct dsn
	dsn := fmt.Sprintf("hdb://%s:%s@%s", user, password, target)

	// create prometheus exporter

	prometheusExporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "hana",
	})

	if err != nil {
		log.Fatalf("Failed to create the Prometheus exporter: %v", err)
	}

	// register prometheus exporter to view

	view.RegisterExporter(prometheusExporter)

	// register default DefaultServerViews to view

	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.Fatalf("Failed to register http default server views for metrics: %v", err)
	}

	// register hana views to view
	if err := view.Register(collector.SYS_M_DisksViews...); err != nil {
		log.Fatalf("Failed to register hana views for metrics: %v", err)
	}

	// start strace and collect data
	ctx, span := trace.StartSpan(context.Background(), "fectch_data")
	defer span.End()
	collector.NEW(ctx, dsn)

	view.SetReportingPeriod(1 * time.Second)
	prometheusExporter.ServeHTTP(w, r)

}

func main() {
	// create prometheus exporter

	// create jaeger exporter

	agentEndpointURI := "10.58.137.243:6831"
	//collectorEndpointURI := "http://10.58.137.243:14268"

	JaegerExporter, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpointURI,
		//	Endpoint:      collectorEndpointURI,
		ServiceName: "hana_exporter",
	})
	if err != nil {
		log.Fatalf("Failed to create the Jaeger exporter: %v", err)
	}
	trace.RegisterExporter(JaegerExporter)

	// apply trace config
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// Ensure that the Prometheus endpoint is exposed for scraping

	// http server mux
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", newHandler)
	mux.HandleFunc("/", homeHandler)
	mux.Handle("/debug/", http.StripPrefix("/debug", zpages.Handler))
	h := &ochttp.Handler{Handler: mux}
	if err := http.ListenAndServe(":9999", h); err != nil {
		log.Fatalf("HTTP server ListenAndServe error: %v", err)
	}

}
