package slappy

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
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

func Handle(writer dns.ResponseWriter, request *dns.Msg) {
	question := request.Question[0]

	message := new(dns.Msg)
	message.SetReply(request)
	message.SetRcode(message, dns.RcodeSuccess)

	full_address := writer.RemoteAddr().String()
	address := strings.Split(full_address, ":")[0]

	if allowed(address) != true {
		msg := fmt.Sprintf("ERROR %s : %s not allowed to talk to slappy", question.Name, address)
		logger.Error(msg)
		message = handle_error(message, writer, "REFUSED")
		respond(message, question, *request, writer)
		return
	}

	logger.Debug(debug_request(*request, question, writer))

	go Stat("query")

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
		if question.Name == conf.Stats_uri {
			message = Stats_dns_message(message, writer)
			logger.Debug("SUCCESS STATS : Sent runtime stats")
			break
		}
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

	serial, err := get_serial(zone_name, conf.Query_dest)
	if err != nil {
		logger.Error(fmt.Sprintf("CREATE ERROR %s : There was a problem querying %s: %s", zone_name, conf.Query_dest, err))
		return handle_error(message, writer, "SERVFAIL")
	}
	if serial != 0 {
		logger.Info(fmt.Sprintf("CREATE SUCCESS %s : zone already exists", zone_name))
		return message
	}

	zone, err := do_axfr(zone_name)
	if len(zone) == 0 || err != nil {
		if err == nil {
			err = errors.New("0 records in AXFR, probably REFUSED")
		}
		msg := fmt.Sprintf("CREATE ERROR %s : there was a problem with the AXFR: %s", zone_name, err)
		logger.Error(msg)
		return handle_error(message, writer, "SERVFAIL")
	}

	output_path := conf.Zone_file_path + strings.TrimSuffix(zone_name, ".")

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

	serial, err := get_serial(zone_name, conf.Query_dest)
	if err != nil {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : There was a problem querying %s: %s", zone_name, conf.Query_dest, err))
		return handle_error(message, writer, "SERVFAIL")
	}
	if serial == 0 {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : zone doesn't exist", zone_name))
		return handle_error(message, writer, "SERVFAIL")
	}

	// Check our master for the SOA of this zone
	master_serial, err := get_serial(zone_name, conf.Master)
	if err != nil {
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : There was a problem querying %s: %s", zone_name, conf.Master, err))
		return handle_error(message, writer, "SERVFAIL")
	}
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
		if err == nil {
			err = errors.New("0 records in AXFR, probably REFUSED")
		}
		logger.Error(fmt.Sprintf("UPDATE ERROR %s : There was a problem with the AXFR: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	output_path := conf.Zone_file_path + strings.TrimSuffix(zone_name, ".")

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

	serial, err := get_serial(zone_name, conf.Query_dest)
	if err != nil {
		logger.Error(fmt.Sprintf("DELETE ERROR %s : There was a problem querying %s: %s", zone_name, conf.Query_dest, err))
		return handle_error(message, writer, "SERVFAIL")
	}
	if serial == 0 {
		logger.Info(fmt.Sprintf("DELETE SUCCESS %s : zone doesn't exist", zone_name))
		return message
	}

	output_path := conf.Zone_file_path + strings.TrimSuffix(zone_name, ".")

	err = rndc("delzone", zone_name, output_path)
	if err != nil {
		logger.Error(fmt.Sprintf("DELETE ERROR %s : problem executing rndc delzone: %s", zone_name, err))
		return handle_error(message, writer, "SERVFAIL")
	}

	logger.Info(fmt.Sprintf("DELETE SUCCESS %s", zone_name))

	return message
}

func rndc(op, zone_name, output_path string) error {
	// Bop the 'attempts' stat
	go Stat("rndc_att")

	cmd := "rndc"
	zone_clause := ""
	args := []string{}
	var err error

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

	if conf.Limit_rndc == false {
		if err := exec.Command(cmd, args...).Run(); err != nil {
			return err
		}
		if op == "delzone" {
			// delete the file
			err := os.Remove(output_path)
			if err != nil {
				return errors.New(fmt.Sprintf("ERROR : Couldn't delete zonefile %s : %s", output_path, err))
			}
		}
		// Bop the stat for this operation
		go Stat(op)

		return nil
	} else {
		// Just used for logs
		rndc_string := fmt.Sprintf("%s %s", op, strings.Join(args, " "))

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
			// Goroutine will wait until it can write to the 'conf.Rndc_counter'
			conf.Rndc_counter <- "rndc"
			// If it took longer than the timeout to write, we will have a message waiting in interrupt
			select {
			case _ = <-interrupt:
				// If there's an interrupt, we ack conf.Rndc_counter and break out without execing our rndc call
				go func() {
					<-conf.Rndc_counter
					// do a debug log to say timeout
					logger.Error("ERROR RNDC: " + rndc_string + " wasn't executed before timeout")
				}()
				// We don't modify err, so Designate will realize we lied when it polls
				return
			default:
				// If there is no interrupt, we continue to our rndc call
			}

			if e := exec.Command(cmd, args...).Run(); e != nil {
				err = e
			}

			if err != nil {
				logger.Debug(fmt.Sprintf("RNDC ERROR: %s failed: %s", rndc_string, err))
			} else {
				logger.Debug(fmt.Sprintf("RNDC SUCCESS: %s completed", rndc_string))
			}

			go func() {
				<-conf.Rndc_counter
				// We ack conf.Rndc_counter once so that another query can have the lock
			}()
			// Since we've finished, we light up the finished channel, so the main function knows
			// we've finished before the timeout
			finished <- "done"
		}()

		select {
		case _ = <-finished:
			// We finished before the timeout

			// Bop the stat for this operation
			go Stat(op)

			if op == "delzone" {
				// delete the file
				err := os.Remove(output_path)
				if err != nil {
					return errors.New(fmt.Sprintf("ERROR : Couldn't delete zonefile %s : %s", output_path, err))
				}
			}
		case <-time.After(conf.Rndc_timeout):
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
	transfer := &dns.Transfer{DialTimeout: conf.Query_timeout, ReadTimeout: conf.Query_timeout}
	if conf.Transfer_source != nil {
		d := net.Dialer{LocalAddr: conf.Transfer_source}
		c, err := d.Dial("tcp", conf.Master)
		if err != nil {
			logger.Debug("AXFR ERROR : problem dialing master")
			return result, err
		}
		dnscon := &dns.Conn{Conn: c}
		transfer = &dns.Transfer{Conn: dnscon, DialTimeout: conf.Query_timeout, ReadTimeout: conf.Query_timeout}
	}

	channel, err := transfer.In(message, conf.Master)
	if err != nil {
		return result, err
	}

	for envelope := range channel {
		result = append(result, envelope.RR...)
	}
	return result, nil
}

func get_serial(zone_name, query_dest string) (uint32, error) {
	var in *dns.Msg
	m := new(dns.Msg)
	m.SetQuestion(zone_name, dns.TypeSOA)

	if conf.Transfer_source != nil {
		d := net.Dialer{LocalAddr: conf.Transfer_source}
		c, err := d.Dial("tcp", query_dest)
		if err != nil {
			logger.Error(fmt.Sprintf("QUERY ERROR : problem dialing query_dest %s", query_dest))
			return 0, err
		}
		co := &dns.Conn{Conn: c}
		co.WriteMsg(m)
		in, err = co.ReadMsg()
		if err != nil {
			logger.Error(fmt.Sprintf("QUERY ERROR : problem querying query_dest %s", query_dest))
			return 0, err
		}
		co.Close()
	} else {
		c := &dns.Client{DialTimeout: conf.Query_timeout, ReadTimeout: conf.Query_timeout}
		if conf.All_tcp == true {
			c.Net = "tcp"
		}
		// _ is query time, might be useful later
		var err error
		in, _, err = c.Exchange(m, query_dest)
		if err != nil {
			logger.Error(fmt.Sprintf("QUERY ERROR : problem querying query_dest %s", query_dest))
			return 0, err
		}
	}
	return serial_query_parse(in), nil
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

	err := ioutil.WriteFile(output_path, []byte(zonefile), 0755)
	if err != nil {
		return err
	}
	return nil
}

func allowed(notifier string) bool {
	if len(conf.Allow_notify) == 0 {
		return true
	}
	for _, ip := range conf.Allow_notify {
		if notifier == ip {
			return true
		}
	}
	return false
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
