package stats

import (
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"runtime"
	"time"

	"github.com/rackerlabs/slappy/config"
	"github.com/rackerlabs/slappy/log"
)

var (
	starttime  time.Time
	stats_chan chan string
	queries    int
	addzones   int
	reloads    int
	delzones   int
	total_rndc int
	rndc_att   int
)

func status_file() {
	conf := config.Conf()
	logger := log.Logger()

	filepath := conf.Status_file
	if filepath == "" {
		logger.Info("Not writing status files")
		return
	}

	err := ioutil.WriteFile(filepath, []byte(""), 0755)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR STATUS FILE : %s", err))
	}

	ticker := time.NewTicker(conf.Status_interval)
	for _ = range ticker.C {
		status := get_status()
		err := ioutil.WriteFile(filepath, []byte(status), 0755)
		if err != nil {
			logger.Error(fmt.Sprintf("ERROR STATUS FILE : %s", err))
		}
		logger.Debug(fmt.Sprintf("SUCCESS STATUS FILE : Wrote %s to %s", status, filepath))
	}
}

func get_status() string {
	// Figure out if we're in an error state
	return "0"
}

func runtime_stats() []string {
	rstats := []string{}

	rstats = append(rstats, fmt.Sprintf("uptime: %.f", time.Now().Sub(starttime).Seconds()))
	rstats = append(rstats, fmt.Sprintf("goroutines: %d", runtime.NumGoroutine()))

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	rstats = append(rstats, fmt.Sprintf("memory: %.3fmb", float64(mem.Alloc)/1000000.0))
	rstats = append(rstats, fmt.Sprintf("nextGC: %.3fmb", float64(mem.NextGC)/1000000.0))

	rstats = append(rstats, fmt.Sprintf("queries: %d", queries))
	rstats = append(rstats, fmt.Sprintf("addzones: %d", addzones))
	rstats = append(rstats, fmt.Sprintf("reloads: %d", reloads))
	rstats = append(rstats, fmt.Sprintf("delzones: %d", delzones))
	rstats = append(rstats, fmt.Sprintf("rndc_attempts: %d", rndc_att))
	rstats = append(rstats, fmt.Sprintf("rndc_success: %d", total_rndc))

	return rstats
}

func Stats_dns_message(message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	conf := config.Conf()

	for _, stat := range runtime_stats() {
		txtRR := new(dns.TXT)
		txtRR.Hdr = dns.RR_Header{Name: conf.Stats_uri, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0}
		txtRR.Txt = []string{stat}
		message.Answer = append(message.Answer, txtRR)
	}

	return message
}

func Stat(stat string) {
	stats_chan <- stat
}

func stats_listener() {
	for {
		select {
		case event := <-stats_chan:
			switch event {
			case "query":
				queries++
			case "addzone":
				addzones++
				total_rndc++
			case "delzone":
				delzones++
				total_rndc++
			case "reload":
				reloads++
				total_rndc++
			case "rndc_att":
				rndc_att++
			default:
			}
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func Init_stats() {
	starttime = time.Now()
	stats_chan = make(chan string, 1000)
	go stats_listener()
	go status_file()
}
