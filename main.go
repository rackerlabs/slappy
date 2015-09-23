package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"github.com/vharitonsky/iniflags"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var (
	debug           *bool
	master          *string
	query_dest      *string
	zone_file_path  *string
	transfer_source *net.TCPAddr
	logfile         *string
	logger          Log
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

	// TODO: allow_notify
	// full_address := writer.RemoteAddr().String()
	// address:= strings.Split(full_address, ":")[0]
	// port:= strings.Split(full_address, ":")[1]

	logger.Debug(debug_request(*request, question, writer))

	switch request.Opcode {
	case dns.OpcodeQuery:
		message = handle_error(message, writer, "REFUSED")
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
			message = handle_error(message, writer, "REFUSED")
		}
	default:
		message = handle_error(message, writer, "REFUSED")
	}

	// Apparently this dns library takes the question out on
	// certain RCodes, like REFUSED, which is not right. So we reinsert it.
	message.Question[0].Name = question.Name
	message.Question[0].Qtype = question.Qtype
	message.Question[0].Qclass = question.Qclass
	message.MsgHdr.Opcode = request.Opcode

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

	serial := get_serial(zone_name)
	if serial != 0 {
		logger.Error(fmt.Sprintf("CREATE ERROR %s : zone already exists", zone_name))
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

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true
	return message
}

func handle_notify(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name)
	if serial == 0 {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : zone doesn't exist", zone_name))
		return handle_error(message, writer, "SERVFAIL")
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : There was a problem with the AXFR: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	// TODO Check the SOA record for zone_name and if it's <= serial
	// don't do the rest of this
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

	logger.Info(fmt.Sprintf("UPDATE SUCCESS %s", zone_name))

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true
	return message
}

func handle_delete(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name)
	if serial == 0 {
		logger.Error(fmt.Sprintf("DELETE ERROR %s : zone doesn't exist", zone_name))
		return message
	}

	err := rndc("delzone", zone_name, "")
	if err != nil {
		logger.Error(fmt.Sprintf("DELETE ERROR %s : problem executing rndc delzone: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	logger.Info(fmt.Sprintf("DELETE SUCCESS %s", zone_name))

	// Send an authoritative answer
	message.MsgHdr.Authoritative = true
	return message
}

func rndc(op, zone_name, output_path string) error {
	cmd := "rndc"
	zone_clause := ""
	args := []string{}

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

	if err := exec.Command(cmd, args...).Run(); err != nil {
		return err
	}

	return nil
}

func do_axfr(zone_name string) ([]dns.RR, error) {
	result := []dns.RR{}
	message := new(dns.Msg)
	message.SetAxfr(zone_name)
	transfer := new(dns.Transfer)
	if transfer_source != nil {
		d := net.Dialer{LocalAddr: transfer_source}
		c, err := d.Dial("tcp", *master)
		if err != nil {
			return result, err
		}
		dnscon := &dns.Conn{Conn: c}
		transfer = &dns.Transfer{Conn: dnscon}
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

func get_serial(zone_name string) uint32 {
	m := new(dns.Msg)
	m.SetQuestion(zone_name, dns.TypeSOA)
	in, err := dns.Exchange(m, *query_dest)
	var serial uint32 = 0
	if err != nil {
		return serial
	}
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

func serve(net string) {
	server := &dns.Server{Addr: ":5358", Net: net}
	dns.HandleFunc(".", handle)
	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("Failed to set up the " + net + "server %s", err.Error()))
	}
}

func listen() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

forever:
	for {
		select {
		case s := <-sig:
			logger.Info(fmt.Sprintf("Signal (%d) received, stopping", s))
			break forever
		}
	}
}

type Log struct {
	Debuglogger    log.Logger
	Infologger     log.Logger
	Warnlogger     log.Logger
	Errorlogger    log.Logger
}

func (l *Log) Debug(line string) {
	if *debug == true {
		l.Debuglogger.Println(line)
	}
}

func (l *Log) Info(line string) {
	l.Infologger.Println(line)
}

func (l *Log) Warn(line string) {
	l.Warnlogger.Println(line)
}

func (l *Log) Error(line string) {
	l.Errorlogger.Println(line)
}

func initLog() {
	var logwriter io.Writer = os.Stdout
	if *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		logwriter = io.MultiWriter(f, os.Stdout)
	}

	d := log.New(logwriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	i := log.New(logwriter, "INFO : ", log.Ldate|log.Ltime|log.Lshortfile)
	c := log.New(logwriter, "WARN : ", log.Ldate|log.Ltime|log.Lshortfile)
	e := log.New(logwriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger = Log{Debuglogger: *d, Infologger: *i, Warnlogger: *c, Errorlogger: *e}
}

func debug_request(request dns.Msg, question dns.Question, writer dns.ResponseWriter) string {
	addr := writer.RemoteAddr().String() // ipaddr string
	s := []string{}
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
	logger.Debug(fmt.Sprintf("master = %s", *master))
	logger.Debug(fmt.Sprintf("query_dest = %s", *query_dest))
	logger.Debug(fmt.Sprintf("zone_file_path = %s", *zone_file_path))
	if transfer_source != nil {
		logger.Debug(fmt.Sprintf("transfer_source = %s", (*transfer_source).String()))
	}
	logger.Debug(fmt.Sprintf("logfile = %s", *logfile))
	logger.Debug("****************CONFIG****************")
}

func main() {
	// Load config
	debug = flag.Bool("debug", false, "enables debug mode")
	logfile = flag.String("log", "", "file for the log, if empty will log only to stdout")
	master = flag.String("master", "", "master to zone transfer from")
	query_dest = flag.String("queries", "", "nameserver to query to grok zone state")
	zone_file_path = flag.String("zone_path", "", "path to write zone files")
	trans_src := flag.String("transfer_source", "", "source IP for zone transfers")
	transfer_source = nil
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	// You can specify an .ini file with the -config
	iniflags.Parse()

	// Parse the transfer_source IP into the proper type
	if *trans_src != "" {
		transfer_source = &net.TCPAddr{IP: net.ParseIP(*trans_src)}
	}

	// Set up logging
	initLog()
	debug_config()

	go serve("tcp")
	logger.Info("slappy started tcp listener on :5358")
	go serve("udp")
	logger.Info("slappy started udp listener on :5358")

	listen()
}
