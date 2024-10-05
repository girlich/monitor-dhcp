package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

type Credential struct {
	Type    string   `yaml:"type"`
	Host    string   `yaml:"host"`
	Command []string `yaml:"command"`
}

type DHCPLease struct {
	ExpirationTime   int64  `yaml:"ExpirationTime"`
	MAC              string `yaml:"MAC"`
	IP               string `yaml:"IP"`
	Hostname         string `yaml:"Hostname"`
	ClientIdentifier string `yaml:"ClientIdentifier"`
}

func dnsmasq_get(credentials *Credential, leases *[]DHCPLease) {
	cmd := exec.Command(credentials.Command[0], credentials.Command[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	buf := bufio.NewReader(stdout)
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		var lease DHCPLease
		fmt.Sscanf(string(line), "%d %s %s %s %s", &lease.ExpirationTime, &lease.MAC, &lease.IP, &lease.Hostname, &lease.ClientIdentifier)
		*leases = append(*leases, lease)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func CollectDHCPLeases(credentials []Credential, DHCPLeases *[]DHCPLease) {
	for i := 0; i < len(credentials); i++ {
		switch credentials[i].Type {
		case "dnsmasq":
			dnsmasq_get(&credentials[i], DHCPLeases)
		default:
			fmt.Fprintf(os.Stderr, "unknown DHCP type: %s\n", credentials[i].Type)
		}
	}
}

type LeaseCollector struct {
	Credentials []Credential
}

var (
	dhcpExpirationTime = prometheus.NewDesc(
		"dhcp_expiration_time",
		"Time, when the lease expires",
		[]string{
			"mac",
			"ip",
			"name"},
		nil,
	)
)

func (lc LeaseCollector) Describe(ch chan<- *prometheus.Desc) {
        prometheus.DescribeByCollect(lc, ch)
}

func (lc LeaseCollector) Collect(ch chan<- prometheus.Metric) {
        var DHCPLeases []DHCPLease
        CollectDHCPLeases(lc.Credentials, &DHCPLeases)

        for _, dl := range DHCPLeases {
                ch <- prometheus.MustNewConstMetric(
                dhcpExpirationTime,
                prometheus.GaugeValue,
                float64(dl.ExpirationTime),
                dl.MAC,
                dl.IP,
                dl.Hostname,
                )
	}
}

func prometheusListen(listen string, credentials []Credential) {
	registry := prometheus.NewRegistry()
	fmt.Println("listen on " + listen)
	wc := LeaseCollector{Credentials: credentials}
	registry.MustRegister(wc)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	log.Fatal(http.ListenAndServe(listen, nil))
}

func main() {
	var listen string
	flag.StringVar(&listen, "listen", "", "thing to listen on (like :1234) for Prometheus requests")
	flag.Parse()

	// Read credentials
	byteValue, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
	}

	var credentials []Credential
	yaml.Unmarshal(byteValue, &credentials)

	if listen == "" {
		var Data []DHCPLease
		CollectDHCPLeases(credentials, &Data)
		DataB, _ := yaml.Marshal(&Data)
		fmt.Println(string(DataB))
	} else {
		prometheusListen(listen, credentials)
	}
}
