DOCKER_TAG := slappy-build

all: dependencies build run

build:
	go build slappy.go

run:
	./slappy -debug

dependencies:
	go get github.com/miekg/dns

test:
	./slappy -debug &
	.venv/bin/python send14.py
	.venv/bin/python sendnotify.py
	pkill slappy


docker-build:
	docker build -t $(DOCKER_TAG) .
	docker run -v `pwd`:/build $(DOCKER_TAG) cp slappy /build
