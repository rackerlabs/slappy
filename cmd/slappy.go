package main

import (
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/rackerlabs/slappy"
)

var builddate = ""
var gitref = ""

func serve(net, ip, port string) {
	logger := slappy.Logger()

	bind := fmt.Sprintf("%s:%s", ip, port)
	server := &dns.Server{Addr: bind, Net: net}

	dns.HandleFunc(".", slappy.Handle)
	logger.Info(fmt.Sprintf("slappy starting %s listener on %s", net, bind))

	err := server.ListenAndServe()
	if err != nil {
		panic(fmt.Sprintf("Failed to set up the "+net+"server %s", err.Error()))
	}
}

func listen() {
	logger := slappy.Logger()

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

func main() {
	// Provide a '--version' flag
	version := flag.Bool("version", false, "prints version information")

	// Set up config
	slappy.Setup_config()
	conf := slappy.Conf()

	// Exit if someone just wants to know version
	if *version == true {
		fmt.Println(fmt.Sprintf("built from %s on %s", gitref, builddate))
		os.Exit(0)
	}

	// Set up logging
	slappy.InitLog(conf.Log_syslog, conf.Logfile, conf.Debug)

	// Debug config
	conf.Print()

	// Init Stats
	slappy.Init_stats()

	go serve("tcp", conf.Bind_address, conf.Bind_port)
	go serve("udp", conf.Bind_address, conf.Bind_port)

	listen()
}
