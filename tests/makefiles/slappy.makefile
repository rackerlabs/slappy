SHELL := /bin/bash

# attempt to find the slappy executable
SLAPPY_EXE=$(shell find "$$GOPATH/bin" -name "slappy")

BIND_TAG=slappy-bind
BIND_CID=$(shell docker ps | grep $(BIND_TAG) | cut -f1 -d' ')
BIND_IP=$(shell docker inspect $(BIND_CID) | jq -r '.[0].NetworkSettings.IPAddress')

help:
	@echo "build        - build the docker bind image"
	@echo "start 		- start the container running bind"
	@echo "stop 		- kill and remove the container"
	@echo "check 		- check that bind is running"
	@echo "clean        - delete the docker bind image"
	@echo "ip           - print bind's ip address to paste into configs"

build:
	cp $(SLAPPY_EXE) docker/slappy
	cd docker/ && docker build -t $(BIND_TAG) -f Dockerfile.slappy .

start:
	docker run --name $(BIND_TAG) -d -t $(BIND_TAG)

stop:
	docker kill $(BIND_TAG) || true
	docker rm -f $(BIND_TAG) || true

clean:
	docker rmi -f $(BIND_TAG) || true

check:
	docker exec $(BIND_TAG) rndc status

ip:
	@echo $(BIND_IP)
