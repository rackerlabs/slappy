# slappy

[![Build Status](https://travis-ci.org/rackerlabs/slappy.svg?branch=master)](https://travis-ci.org/rackerlabs/slappy)

Welcome to the designate-agent, rewritten in Go.

When you install go you'll want to make your tree look like:
```
GOPATH=/home/tim/code/golang
$ (/home/tim/code/golang) tree .
.
├── bin
├── pkg
└── src
    └── github.com
        └── rackerlabs
            └── slappy
```

## Dependencies

slappy uses [glide](https://github.com/Masterminds/glide) to manage
dependencies. If you've got that installed, you can run
`make dependencies` to install a couple things into a `vendor` dir.

## Building

`make build` will do it if you have Go installed. If you have docker, you
can build with `make docker-build` and use that instead.

## Configuration

There's a lot of config toggles that slappy understands:
```shell
$ slappy --help
  -all_tcp
        sends all queries over tcp (default true)
  -allowUnknownFlags
        Don't terminate the app if ini file contains unknown flags.
  -allow_notify string
        comma-separated list of IPs allowed to query slappy
  -bind_address string
        IP to listen on
  -bind_port string
        port to listen on (default "5358")
  -config string
        Path to ini config for using in go flags. May be relative to the current executable path.
  -configUpdateInterval duration
        Update interval for re-reading config file set via -config flag. Zero disables config file re-reading.
  -debug
        enables debug mode
  -dumpflags
        Dumps values for all flags defined in the app into stdout in ini-compatible syntax and terminates the app.
  -limit_rndc
        enables limiting concurrent rndc calls with rndc_timeout, rndc_limit
  -log string
        file for the log, if empty will log only to stdout
  -master string
        master to zone transfer from
  -queries string
        nameserver to query to grok zone state
  -query_timeout int
        seconds before output dns queries timeout from slappy (default 10)
  -rndc_limit int
        number of concurrent rndc calls allowed if limit_rndc=true (default 50)
  -rndc_timeout int
        seconds before waiting rndc call will abort (default 25)
  -stats_uri string
        hostname to dig for to get stats, should be an invalid dns name! (default "/stats.")
  -status_file string
        path to write a status file, empty means no status file
  -status_interval int
        seconds to wait between status file writes (default 60)
  -transfer_source string
        source IP for zone transfers
  -use_syslog
        log only to syslog
  -version
        prints version information
  -zone_path string
        path to write zone files
```

These values can be specified in a config file that's passed to `slappy`
at runtime via `slappy --config slappy.conf`.
```shell
$ cat slappy.conf
[slappy]
debug = true
zone_path = /var/cache/bind
log = slappy.log
all_tcp = true
limit_rndc=true
rndc_limit=1
rndc_timeout=1
```

## Stats

`slappy` keeps various stats internally while it runs. If it's restarted, the stats are reset.

You can view the running tally of stats by doing a ``dig @slappyip /stats``.

```
$ dig @slappyip -p 5358 /stats +short
"uptime: 6650360"
"goroutines: 48"
"memory: 1.113mb"
"nextGC: 4.194mb"
"queries: 3383234"
"addzones: 708573"
"reloads: 456498"
"delzones: 251396"
"rndc_attempts: 1416474"
"rndc_success: 1416467"
```

Metrics are mostly self explanatory:

| metric         | explanation                                                |
| -------------- | ---------------------------------------------------------- |
| uptime         | seconds since slappy was started                           |
| goroutines     | current # of goroutines                                    |
| memory         | current mb of memory usage                                 |
| nextGC         | memory(mb) usage when next garbage collection will occur   |
| queries        | the total number of dns queries slappy has processed       |
| addzones       | the total number of rndc addzones slappy has done          |
| reloads        | the total number of rndc reloads slappy has done           |
| delzones       | the total number of rndc delzones slappy has done          |
| rndc_attempts  | the total number of rndc calls slappy has attempted        |
| rndc_success   | the total number of rndc calls slappy has completed        |
