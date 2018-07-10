package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"github.com/prometheus/common/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace  = "currency"
	api = "http://www.apilayer.net/api/live?access_key=%s&format=1"
)

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

type ExchangeRate struct {
	Success bool `json:"success"`
	Terms string `json:"terms"`
	Privacy string `json:"privacy"`
	Timestamp int64 `json:"timestamp"`
	Source string `json:"source"`
	Quotes map[string]float64 `json:"quotes"`
}

type Exporter struct {
	Metrics map[string]*prometheus.Desc
	APIKey string
}

func NewExporter(apiKey string) (*Exporter, error) {
	return &Exporter{APIKey: apiKey, Metrics: map[string]*prometheus.Desc{}}, nil
}

func (exporter *Exporter) GetExchangeRate()(*ExchangeRate, error) {
	var rates ExchangeRate
	rs, err := http.Get(fmt.Sprintf(api, exporter.APIKey))
	if err != nil {
		return nil, err
	}

	defer rs.Body.Close()

	bodyBytes, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bodyBytes, &rates)
	if err != nil {
		return nil, err
	}

	return &rates, nil
}

func (exporter *Exporter) Describe(ch chan<- *prometheus.Desc) {
	rates, err := exporter.GetExchangeRate()
	if err != nil {
		ch <- nil
	}

	for currency, _ := range rates.Quotes {
		metric := strings.ToLower(currency)
		exporter.Metrics[metric] = prometheus.NewDesc(prometheus.BuildFQName(namespace, "", metric), metric, nil, nil)
		ch <- exporter.Metrics[metric]
	}
}

func (exporter *Exporter) Collect(ch chan<- prometheus.Metric) {
	rates, err := exporter.GetExchangeRate()
	if err != nil {
		ch <- nil
	}
	for quote, value := range rates.Quotes {
		metric := strings.ToLower(quote)
		if _, ok := exporter.Metrics[metric]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics[metric], prometheus.GaugeValue, value)
		}
	}
}

func main() {
	var (
		bind = kingpin.Flag("web.listen-address", "address:port to listen on").Default(":9181").String()
		metrics = kingpin.Flag("web.telemetry-path", "uri path to expose metrics").Default("/metrics").String()
		apiKey = kingpin.Flag("apikey", "api key for APILayer").Required().String()
	)

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	exporter, err := NewExporter(*apiKey)
	if err != nil {
		panic(err)
	}

	prometheus.MustRegister(exporter)

	http.Handle(*metrics, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Currency Exporter</title></head>
             <body>
             <h1>Currency Exporter</h1>
             <p><a href='` + *metrics + `'>Metrics</a></p>
             </body>
             </html>`))
	})


	log.Infoln("Starting HTTP server on", *bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}