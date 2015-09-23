package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

var (
	printf          *bool
	master          *string
	query_dest      *string
	zone_file_path  *string
	transfer_source *net.TCPAddr
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

	// if *printf {
	//     fmt.Println(address + " " + port)
	//     fmt.Printf("Message.opcode: %d\n", request.Opcode)
	//     fmt.Println("Question.name: " + question.Name)
	//     fmt.Printf("Question.Qtype: %d\n", question.Qtype)
	//     fmt.Printf("Question.Qclass: %d\n", question.Qclass)
	// }

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
		fmt.Printf("zone %s already exists\n", zone_name)
		return message
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		fmt.Printf("There was a problem with the AXFR: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	output_path := *zone_file_path + zone_name + "zone"

	err = write_zonefile(zone_name, zone, output_path)
	if err != nil {
		fmt.Printf("There was a problem writing the zone file: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	err = rndc("addzone", zone_name, output_path)
	if err != nil {
		fmt.Printf("There was a problem executing rndc addzone: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	if *printf {
		fmt.Printf("%s created\n", zone_name)
	}
	// Send an authoritative answer
	message.MsgHdr.Authoritative = true
	return message
}

func handle_notify(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name)
	if serial == 0 {
		fmt.Printf("zone %s doesn't exist\n", zone_name)
		return handle_error(message, writer, "SERVFAIL")
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		fmt.Printf("There was a problem with the AXFR: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	// TODO Check the SOA record in 'zone' and if it's <= serial
	// don't do the rest of this
	output_path := *zone_file_path + zone_name + "zone"

	err = write_zonefile(zone_name, zone, output_path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return handle_error(message, writer, "SERVFAIL")
	}

	err = rndc("reload", zone_name, output_path)
	if err != nil {
		fmt.Printf("There was a problem executing rndc reload: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	if *printf {
		fmt.Printf("%s updated\n", zone_name)
	}
	// Send an authoritative answer
	message.MsgHdr.Authoritative = true
	return message
}

func handle_delete(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) *dns.Msg {
	zone_name := question.Name

	serial := get_serial(zone_name)
	if serial == 0 {
		fmt.Printf("zone %s doesn't exist\n", zone_name)
		return message
	}

	err := rndc("delzone", zone_name, "")
	if err != nil {
		fmt.Printf("There was a problem executing rndc delzone: %s\n", err)
		return handle_error(message, writer, "SERVFAIL")
	}

	if *printf {
		fmt.Printf("%s deleted\n", zone_name)
	}
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
		fmt.Fprintln(os.Stderr, err)
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
		fmt.Printf("Failed to set up the "+net+"server %s\n", err.Error())
	}
}

func listen() {
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

forever:
	for {
		select {
		case s := <-sig:
			fmt.Println("Signal (%d) received, stopping\n", s)
			break forever
		}
	}
}

func main() {
	fmt.Println("slappy!\n")

	printf = flag.Bool("debug", false, "print extra info")
	master = flag.String("master", "", "master for axfrs")
	query_dest = flag.String("queries", "", "nameserver to query before operating")
	zone_file_path = flag.String("zone_path", "", "path to write zone files")
	trans_src := flag.String("transfer_source", "", "source IP for zone transfers")
	transfer_source = nil
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	if *trans_src != "" {
		transfer_source = &net.TCPAddr{IP: net.ParseIP(*trans_src)}
	}

	go serve("tcp")
	go serve("udp")

	listen()
}
