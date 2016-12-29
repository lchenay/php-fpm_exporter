package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

const (
	namespace = "fpm" // For Prometheus metrics.
)

var (
	listeningAddress = flag.String("telemetry.address", ":9113", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	fpmScrapeURI   = flag.String("fpm.scrape_uri", "http://localhost/fpm_status", "URI to fpm status page")
	insecure         = flag.Bool("insecure", true, "Ignore server certificate if using https")
)

// Exporter collects fpm stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	URI    string
	mutex  sync.RWMutex
	client *http.Client

	scrapeFailures       prometheus.Counter
	processedConnections *prometheus.Desc
	currentConnections   *prometheus.GaugeVec
}

// NewExporter returns an initialized Exporter.
func NewExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping fpm.",
		}),
		processedConnections: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "connections_processed_total"),
			"Number of connections processed by fpm",
			[]string{"stage"},
			nil,
		),
		currentConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_current",
			Help:      "Number of connections currently processed by fpm",
		},
			[]string{"state"},
		),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
	}
}

// Describe describes all the metrics ever exported by the fpm exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.processedConnections
	e.currentConnections.Describe(ch)
	e.scrapeFailures.Describe(ch)
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	resp, err := e.client.Get(e.URI)
	if err != nil {
		return fmt.Errorf("Error scraping fpm: %v", err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		return fmt.Errorf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	if len(lines) != 15 {
		return fmt.Errorf("Unexpected number of lines in status: %v", lines)
	}

	// Pool name
	//fpm type
	//start time2
	//start since3
	//accepted conn

	if err = e.Extract(4, "accepted conn", "accepted_connection", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(5, "listen queue", "listen_queue", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(6, "max listen queue", "max_listen_queue", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(7, "listen queue len", "listen_queue_length", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(8, "idle processes", "idle_processes", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(9, "active processes", "active_processes", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(10, "total processes", "total_processes", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(11, "max active processes", "max_active_processes", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(12, "max children reached", "max_children_reached", lines) ; err != nil {
		return err;
	}
	if err = e.Extract(13, "slow requests", "slow_request", lines) ; err != nil {
		return err;
	}

	// current connections
	return nil
}

func (e *Exporter) Extract(line int, name string, label string, lines []string) error {
	parts := strings.Split(lines[line], ":")
	if len(parts) != 2 || parts[0] != name {
		return fmt.Errorf("Unexpected line: %s\nExpected: %s", lines[line], name)
	}
	v, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return err
	}
	e.currentConnections.WithLabelValues(label).Set(float64(v))
	return nil
}
// Collect fetches the stats from configured fpm location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping fpm: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	e.currentConnections.Collect(ch)
	return
}

func main() {
	flag.Parse()

	exporter := NewExporter(*fpmScrapeURI)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s", *listeningAddress)
	http.Handle(*metricsEndpoint, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Fpm Exporter</title></head>
			<body>
			<h1>Fpm Exporter</h1>
			<p><a href="` + *metricsEndpoint + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Fatal(http.ListenAndServe(*listeningAddress, nil))
}
