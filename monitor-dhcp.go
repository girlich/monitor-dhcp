package main

import (
  "bufio"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "os"
  "os/exec"

  "gopkg.in/yaml.v2"
)

type Credential struct {
  Type    string   `yaml:"type"`
  Host    string   `yaml:"host"`
  Command []string `yaml:"command"`
}

type DnsmasqLeases struct {
  Time             int
  MAC              string
  IP               string
  Hostname         string
  ClientIdentifier string
}


func dnsmasq_get(credentials *Credential, leases *DnsmasqLeases) {
  cmd := exec.Command(credentials.Command[0], credentials.Command[1:]...)
  stdout, err := cmd.StdoutPipe()
  if err != nil {
    log.Fatal(err)
  }
  cmd.Start()
  buf := bufio.NewReader(stdout)
  for {
    line, _, err := buf.ReadLine()
    if err == io.EOF {
      break
    }
    fmt.Printf(">>%s<<\n", string(line))
  }
}

func main() {
  // Read credentials
  byteValue, err := ioutil.ReadAll(os.Stdin)
  if err != nil {
    fmt.Println(err)
  }

  var credentials []Credential
  yaml.Unmarshal(byteValue, &credentials)

  for i := 0 ; i<len(credentials) ; i++ {
    switch credentials[i].Type {
      case "dnsmasq":
        var Data DnsmasqLeases
        dnsmasq_get(&credentials[i],&Data)
      default:
        fmt.Fprintf(os.Stderr, "unknown DHCP type: %s\n", credentials[i].Type)
    }
  }
}
