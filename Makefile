all: dependencies build run

build:
	go build slappy.go

run:
	./slappy

dependencies:
	go get github.com/miekg/dns
