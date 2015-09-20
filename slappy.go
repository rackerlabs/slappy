package main

import (
    "fmt"
    "flag"
    "github.com/miekg/dns"
    "os"
    "os/signal"
    "strings"
    "syscall"
)

var (
    printf   *bool
    master   *string
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
    message := new(dns.Msg)
    message.SetReply(request)

    // TODO: allow_notify
    full_address := writer.RemoteAddr().String()
    address:= strings.Split(full_address, ":")[0]
    port:= strings.Split(full_address, ":")[1]

    question := request.Question[0]
    if *printf {
        fmt.Println(address + " " + port)
        fmt.Printf("Message.opcode: %d\n", request.Opcode)
        fmt.Println("Question.name: " + question.Name)
        fmt.Printf("Question.Qtype: %d\n", question.Qtype)
        fmt.Printf("Question.Qclass: %d\n", question.Qclass)
    }
    
    switch request.Opcode {
        case dns.OpcodeQuery:
            handle_error(message, writer)
        case dns.OpcodeNotify:
            handle_notify(question, message, writer)
        case CC:
            if question.Qclass == ClassCC {
                switch question.Qtype {
                    case CREATE:
                        handle_create(question, message, writer)
                    case DELETE:
                        handle_delete(question, message, writer)
                    default:
                        handle_error(message, writer)
                }
            } else {
                handle_error(message, writer)
            }
        default:
            handle_error(message, writer)
    }

    writer.WriteMsg(message)
}

func handle_error(message *dns.Msg, writer dns.ResponseWriter) {
    message.SetRcode(message, dns.RcodeRefused)
    writer.WriteMsg(message)
}

func handle_create(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) {
    fmt.Println("rndc addzone")
    writer.WriteMsg(message)
}

func handle_notify(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) {
    fmt.Println("AXFR")
    writer.WriteMsg(message)
}

func handle_delete(question dns.Question, message *dns.Msg, writer dns.ResponseWriter) {
    fmt.Println("rndc delzone")
    writer.WriteMsg(message)
}

func do_axfr(zone_name string) []dns.RR {
    transfer := new(dns.Transfer)
    message := new(dns.Msg)
    message.SetAxfr(zone_name)

    channel, err := transfer.In(message, *master)
    if err != nil {
        fmt.Printf("Error on AXFR %s\n", err.Error())
    }

    result := []dns.RR{}
    for envelope := range channel {
        result = append(result, envelope.RR...)
    }
    return result
}

func serve(net string) {
    server := &dns.Server{Addr: ":8053", Net: net}
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
    flag.Usage = func() {
        flag.PrintDefaults()
    }
    flag.Parse()
    
    go serve("tcp")
    go serve("udp")

    listen()
}
