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

When you clone, you should `git clone --recursive`. If you forgot, no big deal, just run `make dependencies`,
and it'll fill out those submodules for you.

There are a couple of pythonscripts to send some dns packets that the slappy will respond to.
