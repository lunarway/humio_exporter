package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v3"
)

type queryJob struct {
	Id           string
	Timespan     string
	Repo         string
	MetricName   string
	MetricLabels []MetricLabel
}

type queryJobData struct {
	Done            bool
	Events          []map[string]interface{}
	FieldOrder      []string
	MetaData        map[string]interface{}
	ExtraData       map[string]string
	ProcessedEvents int
}

type MetricMap struct {
	Gauges map[string]*prometheus.GaugeVec
}

type YamlConfig struct {
	Queries []struct {
		Query        string        `yaml:"query"`
		Repo         string        `yaml:"repo"`
		Interval     string        `yaml:"interval"`
		MetricName   string        `yaml:"metric_name"`
		MetricLabels []MetricLabel `yaml:"metric_labels"`
	} `yaml:"queries"`
}

type MetricLabel struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

var (
	version            = ""
	supportedFunctions = []string{"_count", "_min", "_max", "_avg", "_rate", "_range", "_stddev", "_sum"}
)

const (
	repoLabel     = "repo"
	intervalLabel = "interval"
)

func main() {
	flags := kingpin.New("humio_exporter", "Humio exporter for Prometheus. Provide your Humio API token and configuration file with queries to expose as Prometheus metrics.")
	configFile := flags.Flag("config", "The humio_exporter configuration file to be used").Required().String()
	baseURL := flags.Flag("humio.url", "Humio base API url").Required().String()
	apiToken := flags.Flag("humio.api-token", "Humio API token").Required().String()
	requestTimeout := flags.Flag("humio.timeout", "Timeout for requests against the Humio API").Default("10").Int()
	listenAddress := flags.Flag("web.listen-address", "Address on which to expose metrics.").Default(":9534").String()
	log.AddFlags(flags)
	flags.HelpFlag.Short('h')
	flags.Version(version)
	kingpin.MustParse(flags.Parse(os.Args[1:]))

	// Parse YAML queries file
	yamlConfig := YamlConfig{}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	yamlFile, err := ioutil.ReadFile(path.Join(currentDir, *configFile))
	if err != nil {
		log.Infof("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal([]byte(yamlFile), &yamlConfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Register the prometheus metrics
	metricMap := MetricMap{
		Gauges: make(map[string]*prometheus.GaugeVec),
	}

	for _, q := range yamlConfig.Queries {
		metricMap.AddGauge(q.MetricName, q.MetricLabels)
	}

	err = metricMap.Register()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "healthy")
	})

	// TODO: Add more logic on when the exporter is actually ready
	// e.g. connection to humio is succesful
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "healthy")
	})

	done := make(chan error, 1)
	go func() {
		log.Infof("Listening on %s", *listenAddress)
		err := http.ListenAndServe(*listenAddress, nil)
		if err != nil {
			done <- err
		}
	}()

	go runAPIPolling(done, *baseURL, *apiToken, yamlConfig, secondDuration(*requestTimeout), metricMap)

	reason := <-done
	if reason != nil {
		log.Errorf("Humio_exporter exited due to error: %v", reason)
		os.Exit(1)
	}
	log.Infof("Humio_exporter exited with exit 0")
}

func runAPIPolling(done chan error, url, token string, yamlConfig YamlConfig, requestTimeout time.Duration, metricMap MetricMap) {
	client := client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		token:   token,
		baseURL: url,
	}

	var jobs []queryJob

	for _, q := range yamlConfig.Queries {
		job, err := client.startQueryJob(q.Query, q.Repo, q.MetricName, q.Interval, "now", q.MetricLabels)
		if err != nil {
			done <- err
			return
		}
		jobs = append(jobs, job)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		for _, job := range jobs {
			client.stopQueryJob(job.Id, job.Repo)
		}
		done <- fmt.Errorf("received os signal '%s'", sig)
	}()

	for {
		for _, job := range jobs {
			poll, err := client.pollQueryJob(job.Id, job.Repo)
			if err != nil {
				done <- err
				return
			}

			var floatValue float64
			for _, f := range supportedFunctions {
				value, ok := poll.Events[0][f]
				if !ok {
					continue
				}
				floatValue, err = parseFloat(value)
				if err != nil {
					done <- err
					return
				}
				break
			}

			if poll.Done {
				metricMap.UpdateMetricValue(job.MetricName, job.Timespan, job.Repo, floatValue, job.MetricLabels)
				if err != nil {
					done <- err
					return
				}
			} else {
				log.Debugf("Skipped value because query isn't done. Timespan: %v, Value: %v", job.Timespan, floatValue)
			}
		}
		time.Sleep(5000 * time.Millisecond)
	}
}

func secondDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

func (m *MetricMap) Register() error {
	for _, v := range m.Gauges {
		err := prometheus.Register(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MetricMap) UpdateMetricValue(metricName, timespan, repo string, value float64, staticLabels []MetricLabel) error {

	labels := make(map[string]string)
	labels[intervalLabel] = timespan
	labels[repoLabel] = repo
	for _, l := range staticLabels {
		labels[l.Key] = l.Value
	}

	gauge := m.Gauges[metricName]
	gauge.With(labels).Set(value)
	return nil
}

func (m *MetricMap) AddGauge(metricName string, staticLabels []MetricLabel) error {
	var labelKeys []string
	labelKeys = append(labelKeys, intervalLabel)
	labelKeys = append(labelKeys, repoLabel)
	for _, l := range staticLabels {
		labelKeys = append(labelKeys, l.Key)
	}

	m.Gauges[metricName] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: "Gauge for humio query",
		}, labelKeys)
	return nil
}
