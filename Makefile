DOCKER_TAG := slappy-build

all: dependencies build run

build:
	go build -o slappy main.go

run:
	./slappy -debug

fmt:
	go fmt main.go

dependencies:
	go get github.com/miekg/dns
	go get github.com/vharitonsky/iniflags

test:
	./slappy -debug &
	.venv/bin/python send14.py
	.venv/bin/python sendnotify.py
	pkill slappy


docker-build:
	docker build -t $(DOCKER_TAG) .
	docker run -v `pwd`:/build $(DOCKER_TAG) cp slappy /build
