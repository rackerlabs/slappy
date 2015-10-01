package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/rackerlabs/dns"
	"github.com/rackerlabs/iniflags"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rackerlabs/slappy/log"
)

var (
	debug           *bool
	logfile         *string
	logger          log.Log
	bind_address    *string
	bind_port       *string
	all_tcp         *bool
	master          *string
	query_dest      *string
	zone_file_path  *string
	query_timeout   time.Duration
	transfer_source *net.TCPAddr
	allow_notify    []string
	limit_rndc      *bool
	rndc_timeout    time.Duration
	rndc_limit       *int
	rndc_counter    chan string
)

// Command and Control OPCODE
const CC = 14

// Private DNS CLASS Uses
const ClassCC = 65280

// Private RR Code Uses
const SUCCESS = 65280
const FAILURE = 65281
const CREATE = 65282
const DELETE = 65283

func handle(writer dns.ResponseWriter, request *dns.Msg) {
	question := request.Question[0]

	message := new(dns.Msg)
	message.SetReply(request)
	message.SetRcode(message, dns.RcodeSuccess)

	full_address := writer.RemoteAddr().String()
	address:= strings.Split(full_address, ":")[0]

	if allowed(address) != true {
		msg := fmt.Sprintf("ERROR %s : %s not allowed to talk to slappy", question.Name, address)
		logger.Error(msg)
		message = handle_error(message, writer, "REFUSED")
		respond(message, question, *request, writer)
		return
	}

	logger.Debug(debug_request(*request, question, writer))

	switch request.Opcode {
	case dns.OpcodeNotify:
		message = handle_notify(question, message, writer)
	case CC:
		if question.Qclass == ClassCC {
			switch question.Qtype {
			case CREATE:
				message = handle_create(question, message, writer)
			case DELETE:
				message = handle_delete(question, message, writer)
			default:
				message = handle_error(message, writer, "REFUSED")
			}
		} else {
			logger.Debug(fmt.Sprintf("ERROR %s : unsupported rrclass %d", question.Name, question.Qclass))
			message = handle_error(message, writer, "REFUSED")
		}
	default:
		logger.Debug(fmt.Sprintf("ERROR %s : unsupported opcode %d", question.Name, request.Opcode))
		message = handle_error(message, writer, "REFUSED")
	}

	respond(message, question, *request, writer)
}

func respond(message *dns.Msg, question dns.Question, request dns.Msg, writer dns.ResponseWriter) {
	// Apparently this dns library takes the question out on
	// certain RCodes, like REFUSED, which is not right. So we reinsert it.
	message.Question[0].Name = question.Name
	message.Question[0].Qtype = question.Qtype
	message.Question[0].Qclass = question.Qclass
	message.MsgHdr.Opcode = request.Opcode

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true

	writer.WriteMsg(message)
}

func handle_error(message *dns.Msg, writer dns.ResponseWriter, op string) *dns.Msg {
	switch op {
	case "REFUSED":
		message.SetRcode(message, dns.RcodeRefused)
	case "SERVFAIL":
		message.SetRcode(message, dns.RcodeServerFailure)
	default:
		message.SetRcode(message, dns.RcodeServerFailure)
	}

	return message
}

func handle_create(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name, *query_dest)
	if serial != 0 {
		logger.Info(fmt.Sprintf("CREATE SUCCESS %s : zone already exists", zone_name))
		return message
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		msg := fmt.Sprintf("CREATE ERROR %s : there was a problem with the AXFR: %s", zone_name, err)
		logger.Error(msg)
		return handle_error(message, writer, "SERVFAIL")
	}

	output_path := *zone_file_path + zone_name + "zone"

	err = write_zonefile(zone_name, zone, output_path)
	if err != nil {
		msg := fmt.Sprintf("CREATE ERROR %s : there was a problem writing the zone file: %s", zone_name, err)
		logger.Error(msg)
		return handle_error(message, writer, "SERVFAIL")
	}

	err = rndc("addzone", zone_name, output_path)
	if err != nil {
		logger.Error(fmt.Sprintf("CREATE ERROR %s : problem executing rndc addzone: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	logger.Info(fmt.Sprintf("CREATE SUCCESS %s", zone_name))

	return message
}

func handle_notify(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name, *query_dest)
	if serial == 0 {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : zone doesn't exist", zone_name))
		return handle_error(message, writer, "SERVFAIL")
	}

	// Check our master for the SOA of this zone
	master_serial := get_serial(zone_name, *master)
	if master_serial == 0 {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : problem with master SOA query", zone_name))
		return handle_error(message, writer, "SERVFAIL")
	}
	if master_serial <= serial {
		logger.Info(fmt.Sprintf("UPDATE SUCCESS %s : already have latest version %d", zone_name, serial))
		return message
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : There was a problem with the AXFR: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	output_path := *zone_file_path + zone_name + "zone"

	err = write_zonefile(zone_name, zone, output_path)
	if err != nil {
		msg := fmt.Sprintf("UPDATE ERROR %s : there was a problem writing the zone file: %s", zone_name, err)
		logger.Error(msg)
		return handle_error(message, writer, "SERVFAIL")
	}

	err = rndc("reload", zone_name, output_path)
	if err != nil {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : there was a problem executing rndc reload: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	logger.Info(fmt.Sprintf("UPDATE SUCCESS %s serial %d", zone_name, master_serial))

	return message
}

func handle_delete(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name, *query_dest)
	if serial == 0 {
		logger.Info(fmt.Sprintf("DELETE SUCCESS %s : zone doesn't exist", zone_name))
		return message
	}

	err := rndc("delzone", zone_name, "")
	if err != nil {
		logger.Error(fmt.Sprintf("DELETE ERROR %s : problem executing rndc delzone: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	// TODO: Delete the zonefile maybe?

	logger.Info(fmt.Sprintf("DELETE SUCCESS %s", zone_name))

	return message
}

func rndc(op, zone_name, output_path string) error {
	cmd := "rndc"
	zone_clause := ""
	args := []string{}
	var err error;

	switch op {
	case "addzone":
		zone_clause = fmt.Sprintf("{ type master; file \"%s\"; };", output_path)
		args = []string{"-s", "127.0.0.1", "-p", "953", op, strings.TrimSuffix(zone_name, "."), zone_clause}
	case "delzone":
		args = []string{"-s", "127.0.0.1", "-p", "953", op, strings.TrimSuffix(zone_name, ".")}
	case "reload":
		args = []string{"-s", "127.0.0.1", "-p", "953", op, strings.TrimSuffix(zone_name, ".")}
	default:
		return errors.New("Invalid RNDC command")
	}

	if *limit_rndc == false {
		if e := exec.Command(cmd, args...).Run(); e != nil {
			return err
		}
		return nil
	} else {
		rndc_string := op+" "+strings.Join(args, " ")

		// finished will get filled if the rndc call finishes before the timeout
		finished := make(chan string, 1)
		// if finished doesn't get filled, interrupt will send an interrupt message
		// to stop the rndc call from executing
		interrupt := make(chan string, 1)
		// timeout is the amount of time we're going to wait synchronously in a
		// goroutine handling a query. We'd like to wait until the status of the
		// rndc comes back, so we can tell Designate. But if it goes too long, we will
		// just return error so Designate knows to try again

		// spawn a new Goroutine, this immediately jumps down to the select statement after the
		// function call and starts waiting on the timeout, or the ack that the call finished
		go func() {
			// Goroutine will wait until it can write to the 'rndc_counter'
			rndc_counter <- "rndc"
			// If it took longer than the timeout to write, we will have a message waiting in interrupt
			select {
			case _ = <-interrupt:
				// If there's an interrupt, we ack rndc_counter and break out without execing our rndc call
				go func() {
					<-rndc_counter
					// do a debug log to say timeout
					logger.Error("ERROR RNDC: "+rndc_string+" wasn't executed before timeout")
				}()
				// We don't modify err, so Designate will realize we lied when it polls
				return
			default:
				// If there is no interrupt, we continue to our rndc call
			}

			if e := exec.Command(cmd, args...).Run(); e != nil {
				err = e
			}

			go func() {
				<-rndc_counter
				// We ack rndc_counter once so that another query can have the lock
				logger.Debug("RNDC SUCCESS: "+rndc_string+" completed")
			}()
			// Since we've finished, we light up the finished channel, so the main function knows
			// we've finished before the timeout
			finished <- "done"
		}()

		select {
		case _ = <-finished:
			// We finished before the timeout
		case <-time.After(rndc_timeout):
			// We have timed out, throw away the rndc call by interupting the goroutine above
			// Spawn a GoRoutine for the interrupt so that we can return this function
			// There's a small amount of time between realizing the timeout, and before this
			// goroutine does it's thing. It's possible that the rndc call might execute
			// This is ok, because when Designate goes to fix up the change, slappy will promptly
			// return that this is all done
			go func() { interrupt <- "interrupt" }()
		}

		return err
	}
}

func do_axfr(zone_name string) ([]dns.RR, error) {
	result := []dns.RR{}
	message := new(dns.Msg)
	message.SetAxfr(zone_name)
	transfer := &dns.Transfer{DialTimeout: query_timeout, ReadTimeout: query_timeout}
	if transfer_source != nil {
		d := net.Dialer{LocalAddr: transfer_source}
		c, err := d.Dial("tcp", *master)
		if err != nil {
			logger.Debug("AXFR ERROR : problem dialing master")
			return result, err
		}
		dnscon := &dns.Conn{Conn: c}
		transfer = &dns.Transfer{Conn: dnscon, DialTimeout: query_timeout, ReadTimeout: query_timeout}
	}

	channel, err := transfer.In(message, *master)
	if err != nil {
		return result, err
	}

	for envelope := range channel {
		result = append(result, envelope.RR...)
	}
	return result, nil
}

func get_serial(zone_name, query_dest string) uint32 {
	var serial uint32 = 0
	var in *dns.Msg;

	m := new(dns.Msg)
	m.SetQuestion(zone_name, dns.TypeSOA)

	if transfer_source != nil {
		d := net.Dialer{LocalAddr: transfer_source}
		c, err := d.Dial("tcp", query_dest)
		if err != nil {
			logger.Debug(fmt.Sprintf("QUERY ERROR : problem dialing query_dest %s", query_dest))
			return 0
		}
		co := &dns.Conn{Conn: c}
		co.WriteMsg(m)
		in, err = co.ReadMsg()
		if err != nil {
			logger.Debug(fmt.Sprintf("QUERY ERROR : problem querying query_dest %s", query_dest))
			return 0
		}
		co.Close()
	} else {
		c := &dns.Client{DialTimeout: query_timeout, ReadTimeout: query_timeout}
		if *all_tcp == true { c.Net = "tcp" }
		// _ is query time, might be useful later
		var err error;
		in, _, err = c.Exchange(m, query_dest)
		if err != nil {
			logger.Debug(fmt.Sprintf("QUERY ERROR : problem querying query_dest %s", query_dest))
			return serial
		}
	}
	return serial_query_parse(in)
}

func serial_query_parse(in *dns.Msg) uint32 {
	var serial uint32 = 0
	if in.Rcode != dns.RcodeSuccess {
		return serial
	}
	if rr, ok := in.Answer[0].(*dns.SOA); ok {
		serial = rr.Serial
	}
	return serial
}

func write_zonefile(zone_name string, rrs []dns.RR, output_path string) error {
	lines := []string{}
	for _, rr := range rrs {
		lines = append(lines, dns.RR.String(rr), "\n")
	}
	zonefile := strings.Join(lines, "")

	f, err := os.Create(output_path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(zonefile)
	if err != nil {
		return err
	}
	f.Sync()
	return nil
}

func allowed(notifier string) bool {
	if len(allow_notify) == 0 {
		return true
	}
	for _, ip := range allow_notify {
		if notifier == ip {
			return true
		}
	}
	return false
}

func serve(net, ip, port string) {
	bind := fmt.Sprintf("%s:%s", ip, port)
	server := &dns.Server{Addr: bind, Net: net}
	dns.HandleFunc(".", handle)
	logger.Info(fmt.Sprintf("slappy starting %s listener on %s", net, bind))
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("Failed to set up the "+net+"server %s", err.Error()))
	}
}

func listen() {
	siq_quit := make(chan os.Signal)
	signal.Notify(siq_quit, syscall.SIGINT, syscall.SIGTERM)
	sig_stat := make(chan os.Signal)
	signal.Notify(sig_stat, syscall.SIGUSR1)

forever:
	for {
		select {
		case s := <-siq_quit:
			logger.Info(fmt.Sprintf("Signal (%d) received, stopping", s))
			break forever
		case _ = <-sig_stat:
			logger.Info(fmt.Sprintf("Goroutines: %d", runtime.NumGoroutine()))
		}
	}
}

func debug_request(request dns.Msg, question dns.Question, writer dns.ResponseWriter) string {
	addr := writer.RemoteAddr().String() // ipaddr string
	s := []string{}
	// TODO: put tcp/udp in here
	s = append(s, fmt.Sprintf("Received request from %s ", addr))
	s = append(s, fmt.Sprintf("for %s ", question.Name))
	s = append(s, fmt.Sprintf("opcode: %d ", request.Opcode))
	s = append(s, fmt.Sprintf("rrtype: %d ", question.Qtype))
	s = append(s, fmt.Sprintf("rrclass: %d ", question.Qclass))
	return strings.Join(s, "")
}

func debug_config() {
	logger.Debug("****************CONFIG****************")
	logger.Debug(fmt.Sprintf("debug = %t", *debug))
	logger.Debug(fmt.Sprintf("log = %s", *logfile))
	logger.Debug(fmt.Sprintf("bind_address = %s", *bind_address))
	logger.Debug(fmt.Sprintf("bind_port = %s", *bind_port))
	logger.Debug(fmt.Sprintf("all_tcp = %t", *all_tcp))
	logger.Debug(fmt.Sprintf("master = %s", *master))
	logger.Debug(fmt.Sprintf("query_dest = %s", *query_dest))
	logger.Debug(fmt.Sprintf("zone_file_path = %s", *zone_file_path))
	logger.Debug(fmt.Sprintf("query_timeout = %s", query_timeout))
	logger.Debug(fmt.Sprintf("limit_rndc = %t", *limit_rndc))
	logger.Debug(fmt.Sprintf("rndc_timeout = %s", rndc_timeout))
	logger.Debug(fmt.Sprintf("rndc_limit = %d", *rndc_limit))
	if transfer_source != nil {
		logger.Debug(fmt.Sprintf("transfer_source = %s", (*transfer_source).String()))
	}
	logger.Debug(fmt.Sprintf("allow_notify = %s", allow_notify))
	logger.Debug("****************CONFIG****************")
}

func main() {
	// Load config
	debug = flag.Bool("debug", false, "enables debug mode")
	logfile = flag.String("log", "", "file for the log, if empty will log only to stdout")

	bind_address = flag.String("bind_address", "", "IP to listen on")
	bind_port = flag.String("bind_port", "5358", "port to listen on")
	all_tcp = flag.Bool("all_tcp", true, "sends all queries over tcp")

	master = flag.String("master", "", "master to zone transfer from")
	query_dest = flag.String("queries", "", "nameserver to query to grok zone state")
	zone_file_path = flag.String("zone_path", "", "path to write zone files")
	query_timeout_raw := flag.Int("query_timeout", 10, "seconds before output dns queries timeout from slappy")

	transfer_source_raw := flag.String("transfer_source", "", "source IP for zone transfers")
	transfer_source = nil
	allow_notify_raw := flag.String("allow_notify", "", "comma-separated list of IPs allowed to query slappy")
	allow_notify = []string{}

	limit_rndc = flag.Bool("limit_rndc", false, "enables limiting concurrent rndc calls with rndc_timeout, rndc_limit")
	rndc_timeout_raw := flag.Int("rndc_timeout", 25, "seconds before waiting rndc call will abort")
	rndc_limit = flag.Int("rndc_limit", 50, "number of concurrent rndc calls allowed if limit_rndc=true")

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
	query_timeout = time.Duration(*query_timeout_raw) * time.Second
	rndc_timeout = time.Duration(*rndc_timeout_raw) * time.Second

	// Set up rndc rate limiter
	if *limit_rndc == true { rndc_counter = make(chan string, *rndc_limit) }

	// Set up logging
	logger = log.InitLog(*logfile, *debug)
	debug_config()

	go serve("tcp", *bind_address, *bind_port)
	go serve("udp", *bind_address, *bind_port)

	listen()
}
