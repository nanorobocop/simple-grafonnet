package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/go-pkgz/lgr"
	jsonnet "github.com/google/go-jsonnet"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	metricsURL = flag.String("url", "", "URL to metrics endpoint")
	debug      = flag.Bool("debug", false, "Debug logging")
)

type App struct{ log *lgr.Logger }

func main() {
	flag.Parse()

	logOptions := []lgr.Option{lgr.Out(os.Stderr)}
	if *debug {
		logOptions = append(logOptions, lgr.Debug)
	}

	log := lgr.New(logOptions...)

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
	metricsList := app.buildMetricsList(metrics)
	for _, m := range metricsList {
		log.Logf("DEBUG %+v", m)
	}
	if err != nil {
		log.Logf("FATAL Failed to build metrics list: %+v", err)
	}

	global, err := app.buildGlobalSetting()
	if err != nil {
		log.Logf("FATAL Failed to build global dashboard settings: %+v", err)
	}

	log.Logf("INFO Generating dashboard")
	vm := jsonnet.MakeVM()

	importer := &jsonnet.FileImporter{
		JPaths: []string{"grafonnet-lib"},
	}
	vm.Importer(importer)

	metricsListEncoded, err := json.Marshal(metricsList)
	if err != nil {
		log.Logf("FATAL Failed to marshal metrics list: %+v", err)
	}
	vm.TLACode("metrics", string(metricsListEncoded))
	vm.TLACode("global", global)

	jsonStr, err := vm.EvaluateFile("dashboard.jsonnet")
	if err != nil {
		log.Logf("FATAL Failed to generate dashboard: %+v", err)
	}

	fmt.Println(jsonStr)

	log.Logf("Dashboard generated and printed to stdout!")
}

type Metric struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Expr   string `json:"expr"`
	Format string `json:"format"`
}

type Global struct {
	Datasource string `json:"datasource"`
}

type Metrics []Metric

func (metrics Metrics) NamesSlice() []string {
	slice := []string{}
	for _, m := range metrics {
		slice = append(slice, m.Name)
	}
	return slice
}

type ByName []Metric

func (n ByName) Len() int           { return len(n) }
func (n ByName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByName) Less(i, j int) bool { return n[i].Name < n[j].Name }

func (app *App) buildMetricsList(metrics map[string]*dto.MetricFamily) []Metric {
	metricsList := Metrics{}
	for k, v := range metrics {
		name := k
		if v.Name != nil {
			name = *v.Name
		}

		title := k
		if v.Help != nil {
			title = *v.Help
			title = strings.TrimSuffix(title, ".")
		}
		metric := Metric{
			Title:  title,
			Name:   name,
			Format: "short",
		}
		switch *v.Type {
		case dto.MetricType_COUNTER:
			metric.Expr = fmt.Sprintf("rate(%s[5m])", k)
		case dto.MetricType_GAUGE:
			metric.Expr = k
		case dto.MetricType_HISTOGRAM:
			labels := app.getMetricLabels(v)
			labels = append(labels, "le")
			metric.Expr = fmt.Sprintf("histogram_quantile(0.95, sum(rate(%s_bucket[5m])) by (%s))", k, strings.Join(labels, ","))
		case dto.MetricType_SUMMARY:
			metric.Expr = fmt.Sprintf("rate(%s_sum[5m]) / rate(%s_count[5m])", k, k)
		}

		metricsList = append(metricsList, metric)
	}

	sort.Sort(ByName(metricsList))

	group(metricsList.NamesSlice())

	return metricsList
}

func (app *App) getMetricLabels(metrics *dto.MetricFamily) []string {
	labels := map[string]struct{}{} // use map to avoid duplicates

	for _, metric := range metrics.Metric {
		labelPairs := metric.Label
		for _, l := range labelPairs {
			if l == nil {
				continue
			}
			labels[*l.Name] = struct{}{}
		}
		break
	}

	labelsSlice := []string{}
	for k := range labels {
		labelsSlice = append(labelsSlice, k)
	}

	return labelsSlice
}

func (app *App) buildGlobalSetting() (string, error) {
	glob := Global{
		Datasource: "Prometheus",
	}

	bytes, err := json.Marshal(glob)
	return string(bytes), err
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

// Group does very simple grouping.
// Assuming strings are sorted, it distinguish group and subgroups separated by "_" underscore.
// Idea of grouping is to have metrics grouped by namespace and subsystem.
// Since namespace or system itself could have underscores as its part, splitting is not correct in this case.
// See https://github.com/prometheus/client_golang/blob/0400fc44d42dd0bca7fb16e87ea0313bb2eb8c53/prometheus/metric.go#L68-L72
func group(strs []string) map[string]map[string][]string {
	// groups = map of group of subgroup of strings
	groups := map[string]map[string][]string{}
	group, subgroup := "", ""
	for _, str := range strs {
		elems := strings.Split(str, "_")
		if group != elems[0] {
			if len(elems) > 1 {
				subgroup = elems[1]
			} else {
				subgroup = elems[0]
			}

			group = elems[0]
			groups[group] = map[string][]string{subgroup: []string{str}}
		} else if len(elems) > 1 && subgroup != elems[1] {
			subgroup = elems[1]
			groups[group][subgroup] = []string{str}
		} else {
			groups[group][subgroup] = append(groups[group][subgroup], str)
		}
	}
	return groups
}
