package config

import (
	"flag"
	"fmt"
	"github.com/rackerlabs/iniflags"
	"net"
	"strings"
	"time"

	"github.com/rackerlabs/slappy/log"

)

var conf Config;

type Config struct {
	Debug           bool
	Logfile         string
	Logger          log.Log
	Bind_address    string
	Bind_port       string
	All_tcp         bool
	Master          string
	Query_dest      string
	Zone_file_path  string
	Query_timeout   time.Duration
	Transfer_source *net.TCPAddr
	Allow_notify    []string
	Limit_rndc      bool
	Rndc_timeout    time.Duration
	Rndc_limit      int
	Rndc_counter    chan string
}

// These vars are necessary because the actual values in the `flag.x` don't
// get populated until you call iniflags.Parse() and these require additional processing
var (
	transfer_source *net.TCPAddr
	rndc_counter    chan string
)

func Setup_config() {
	// Load config
	debug := flag.Bool("debug", false, "enables debug mode")
	logfile := flag.String("log", "", "file for the log, if empty will log only to stdout")

	bind_address := flag.String("bind_address", "", "IP to listen on")
	bind_port := flag.String("bind_port", "5358", "port to listen on")
	all_tcp := flag.Bool("all_tcp", true, "sends all queries over tcp")

	master := flag.String("master", "", "master to zone transfer from")
	query_dest := flag.String("queries", "", "nameserver to query to grok zone state")
	zone_file_path_raw := flag.String("zone_path", "", "path to write zone files")
	query_timeout_raw := flag.Int("query_timeout", 10, "seconds before output dns queries timeout from slappy")

	transfer_source_raw := flag.String("transfer_source", "", "source IP for zone transfers")
	transfer_source = nil
	allow_notify_raw := flag.String("allow_notify", "", "comma-separated list of IPs allowed to query slappy")
	allow_notify := []string{}

	limit_rndc := flag.Bool("limit_rndc", false, "enables limiting concurrent rndc calls with rndc_timeout, rndc_limit")
	rndc_timeout_raw := flag.Int("rndc_timeout", 25, "seconds before waiting rndc call will abort")
	rndc_limit := flag.Int("rndc_limit", 50, "number of concurrent rndc calls allowed if limit_rndc=true")

	flag.Usage = func() {
		flag.PrintDefaults()
	}
	// You can specify an .ini file with the -config
	iniflags.Parse()

	// Parse the transfer_source IP into the proper type
	if *transfer_source_raw != "" {
		transfer_source = &net.TCPAddr{IP: net.ParseIP(*transfer_source_raw)}
	}
	if *allow_notify_raw != "" {
		for _, ip := range strings.Split((*allow_notify_raw), ",") {
			allow_notify = append(allow_notify, strings.TrimSpace(ip))
		}
	}
	query_timeout := time.Duration(*query_timeout_raw) * time.Second
	rndc_timeout := time.Duration(*rndc_timeout_raw) * time.Second

	zone_file_path := *zone_file_path_raw
	if !strings.HasSuffix(zone_file_path, "/") {
		zone_file_path = zone_file_path + "/"
	}

	// Set up rndc rate limiter
	if *limit_rndc == true { rndc_counter = make(chan string, *rndc_limit) }

	conf = Config{
		Debug: *debug,
		Logfile: *logfile,
		Bind_address: *bind_address,
		Bind_port: *bind_port,
		All_tcp: *all_tcp,
		Master: *master,
		Query_dest: *query_dest,
		Zone_file_path: zone_file_path,
		Query_timeout: query_timeout,
		Transfer_source: transfer_source,
		Allow_notify: allow_notify,
		Limit_rndc: *limit_rndc,
		Rndc_timeout: rndc_timeout,
		Rndc_limit: *rndc_limit,
		Rndc_counter: rndc_counter,
	}
}

func (c *Config) Print(logger log.Log) {
	logger.Debug("****************CONFIG****************")
	logger.Debug(fmt.Sprintf("debug = %t", c.Debug))
	logger.Debug(fmt.Sprintf("log = %s", c.Logfile))
	logger.Debug(fmt.Sprintf("bind_address = %s", c.Bind_address))
	logger.Debug(fmt.Sprintf("bind_port = %s", c.Bind_port))
	logger.Debug(fmt.Sprintf("all_tcp = %t", c.All_tcp))
	logger.Debug(fmt.Sprintf("master = %s", c.Master))
	logger.Debug(fmt.Sprintf("query_dest = %s", c.Query_dest))
	logger.Debug(fmt.Sprintf("zone_file_path = %s", c.Zone_file_path))
	logger.Debug(fmt.Sprintf("query_timeout = %s", c.Query_timeout))
	logger.Debug(fmt.Sprintf("limit_rndc = %t", c.Limit_rndc))
	logger.Debug(fmt.Sprintf("rndc_timeout = %s", c.Rndc_timeout))
	logger.Debug(fmt.Sprintf("rndc_limit = %d", c.Rndc_limit))
	if c.Transfer_source != nil {
		logger.Debug(fmt.Sprintf("transfer_source = %s", (c.Transfer_source).String()))
	}
	logger.Debug(fmt.Sprintf("allow_notify = %s", c.Allow_notify))
	logger.Debug("****************CONFIG****************")
}

func Conf() Config {
	return conf
}