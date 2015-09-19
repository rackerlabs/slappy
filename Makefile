all: dependencies build run

build:
	go build slappy.go

run:
	./slappy

dependencies:
	go get github.com/miekg/dns

test:
	./slappy &
	.venv/bin/python send14.py
	.venv/bin/python sendnotify.py
	pkill slappy
