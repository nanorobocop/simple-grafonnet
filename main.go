package main

import (
	"flag"
	"net/http"

	"github.com/go-pkgz/lgr"
	jsonnet "github.com/google/go-jsonnet"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	metricsURL = flag.String("url", "", "URL to metrics endpoint")
)

type App struct{ log *lgr.Logger }

func main() {
	flag.Parse()

	log := lgr.New(lgr.Msec, lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

	app := &App{log: log}

	log.Logf("INFO Downloading metrics from endpoint: %+v", *metricsURL)
	resp, err := http.Get(*metricsURL)
	if err != nil {
		log.Logf("FATAL Failed to parse URL: %+v", err)
	}
	defer resp.Body.Close()

	log.Logf("INFO Parsing metrics data")
	parser := expfmt.TextParser{}
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		log.Logf("FATAL Failed to parse metrics: %+v", err)
	}

	app.printMetricsStat(metrics)

	log.Logf("INFO Importing libraries")
	importer := &jsonnet.FileImporter{
		JPaths: []string{"grafonnet-lib"},
	}

	log.Logf("INFO Making Jsonnet VM")
	vm := jsonnet.MakeVM()
	vm.Importer(importer)

	log.Logf("INFO Generating dashboard")
	jsonStr, err := vm.EvaluateFile("dashboard.jsonnet")
	if err != nil {
		log.Logf("FATAL Failed to generate dashboard: %+v", err)
	}

	log.Logf("Dashboard: %+v", jsonStr)
}

func (app *App) printMetricsStat(metrics map[string]*dto.MetricFamily) {
	mtype := map[string]int{
		"counter":   0,
		"gauge":     0,
		"summary":   0,
		"untyped":   0,
		"histogram": 0,
	}
	for _, v := range metrics {
		switch *v.Type {
		case dto.MetricType_COUNTER:
			mtype["counter"]++
		case dto.MetricType_GAUGE:
			mtype["gauge"]++
		case dto.MetricType_SUMMARY:
			mtype["summary"]++
		case dto.MetricType_UNTYPED:
			mtype["untyped"]++
		case dto.MetricType_HISTOGRAM:
			mtype["histogram"]++
		}
	}
	for k, v := range mtype {
		app.log.Logf("INFO Found metrics of type %s: %d", k, v)
	}
}
