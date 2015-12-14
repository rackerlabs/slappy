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
        ├── miekg
        │   └── dns
        │       ├── AUTHORS
        │       ├── clientconfig.go
                .......
        │       └── zscan_rr.go
        └── pglbutt
            └── slappy
                ├── Makefile
                ├── README.md
                ├── send14.py
                ├── sendnotify.py
                └── slappy.go
```

Makefile will get you started.

There are a couple of pythonscripts to send some dns packets that the slappy will respond to.
