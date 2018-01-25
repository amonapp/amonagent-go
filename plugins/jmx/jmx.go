package jmx

import (
	"encoding/json"
	//"sync"

	log "github.com/Sirupsen/logrus"
	//"github.com/amonapp/amonagent/internal/util"
	"github.com/amonapp/amonagent/plugins"
	"strconv"
	"os"
	"os/exec"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// JMX - XXX
type JMX struct {
	Config Config
}

// PerformanceStruct - XXX
type PerformanceStruct struct {
	Gauges map[string]interface{} `json:"gauges"`
}

// Start - XXX
func (c *JMX) Start() error { return nil }

// Stop - XXX
func (c *JMX) Stop() {}

// Description - XXX
func (c *JMX) Description() string {
	return "Collects data from Java Applications"
}

var sampleConfig = `
#   Available config options:
#
#    [
#       {
#         "name": "Application",
#         "hostName": "localhost",
#         "port": 1234"
#       }
#    ]
#
# Config location: /etc/opt/amonagent/plugins-enabled/jmx.conf
`

// SampleConfig - XXX
func (c *JMX) SampleConfig() string {
	return sampleConfig
}

type Endpoint struct {
	Name string `json:"name"`
	HostName string `json:"hostName"`
	Port int `json:"port"`
}

type MJBJson struct {
	ThreadCount int64 `json:"java.lang:type=Threading ThreadCount"`
	DaemonThreadCount int64 `json:"java.lang:type=Threading DaemonThreadCount"`
	HeapMemoryUsageMax int64 `json:"java.lang:type=Memory HeapMemoryUsage max"`
	HeapMemoryUsageInit int64 `json:"java.lang:type=Memory HeapMemoryUsage init"`
	HeapMemoryUsageCommitted int64 `json:"java.lang:type=Memory HeapMemoryUsage committed"`
	HeapMemoryUsageUsed int64 `json:"java.lang:type=Memory HeapMemoryUsage used"`
	NonHeapMemoryUsageInit int64 `json:"java.lang:type=Memory NonHeapMemoryUsage init"`
	NonHeapMemoryUsageCommitted int64 `json:"java.lang:type=Memory NonHeapMemoryUsage committed"`
	NonHeapMemoryUsageUsed int64 `json:"java.lang:type=Memory NonHeapMemoryUsage used"`
}

// Config - XXX
type Config struct {
	Endpoints []Endpoint `json:"endpoints"`
}

// SetConfigDefaults - XXX
func (c *JMX) SetConfigDefaults() error {

	// Commands already set. For example - in the test suite
	if len(c.Config.Endpoints) > 0 {
		return nil
	}

	configFile, err := plugins.ReadPluginConfig("jmx")
	if err != nil {
		log.WithFields(log.Fields{
			"plugin": "jmx",
			"error":  err,
		}).Error("Can't read config file")

		return err
	}

	var Endpoints []Endpoint

	if e := json.Unmarshal(configFile, &Endpoints); e != nil {
		log.WithFields(log.Fields{"plugin": "jmx", "error": e.Error()}).Error("Can't decode JSON file")
		return e
	}

	c.Config.Endpoints = Endpoints

	return nil
}

// Collect - XXX
func (c *JMX) Collect() (interface{}, error) {
	c.SetConfigDefaults()
	PerformanceStruct := PerformanceStruct{}
	gauges := map[string]interface{}{

	}

	for _, v := range c.Config.Endpoints {
		var rawJson, err= runJar(v.HostName, v.Port)
		if err != nil {
			continue
		}
		var data MJBJson

		if e := json.Unmarshal([]byte(rawJson), &data); e != nil {
			log.WithFields(log.Fields{"plugin": "jmx", "error": e.Error()}).Error("Can't decode jmx response")
			continue
		}
		gauges[v.Name + "_jmx_threads.threadCount"] = data.ThreadCount
		gauges[v.Name + "_jmx_threads.daemonThreadCount"] = data.DaemonThreadCount
		gauges[v.Name + "_jmx_heapMemory.committed"] = data.HeapMemoryUsageCommitted
		gauges[v.Name + "_jmx_heapMemory.init"] = data.HeapMemoryUsageInit
		gauges[v.Name + "_jmx_heapMemory.max"] = data.HeapMemoryUsageMax
		gauges[v.Name + "_jmx_heapMemory.used"] = data.HeapMemoryUsageUsed
		gauges[v.Name + "_jmx_nonHeapMemory.committed"] = data.NonHeapMemoryUsageCommitted
		gauges[v.Name + "_jmx_nonHeapMemory.init"] = data.NonHeapMemoryUsageInit
		gauges[v.Name + "_jmx_nonHeapMemory.used"] = data.NonHeapMemoryUsageUsed
	}

	PerformanceStruct.Gauges = gauges

	return PerformanceStruct, nil
}

func init() {
	plugins.Add("jmx", func() plugins.Plugin {
		return &JMX{}
	})
}

// RunJar runs the embedded jar returning the output from STDOUT
func runJar(host string, port int) (string, error) {
	e := ensureJarExists()
	if e != nil {
		return "", e
	}
	nport := strconv.Itoa(port)

	cmd := exec.Command("java", "-jar", JarFile, host, nport)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("%s %s", err.Error(), out.String())
	}

	return out.String(), nil
}

func ensureJarExists() (error) {
	_, err := os.Stat(JarFile)
	if err != nil {
		data, err := Asset("data/mjb.jar")
		if err != nil {
			return err
		}
		permissions := os.FileMode(644)
		os.Mkdir(filepath.Dir(JarFile), permissions)
		err = ioutil.WriteFile(JarFile, data, permissions)
		if err != nil {
			return err
		}
	}
	return nil
}