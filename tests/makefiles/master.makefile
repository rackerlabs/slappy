SHELL := /bin/bash

BIND_TAG=slappy-bind-master
BIND_CID=$(shell docker ps | grep "$(BIND_TAG) " | cut -f1 -d' ')
BIND_IP=$(shell if [ ! -z "$(BIND_CID)" ]; then docker inspect --format '{{ .NetworkSettings.IPAddress }}' $(BIND_CID); fi)

help:
	@echo "build        - build the docker bind image"
	@echo "start        - start the container running bind"
	@echo "stop         - kill and remove the container"
	@echo "check        - check that bind is running"
	@echo "clean        - delete the docker bind image"
	@echo "ip           - print bind's ip address to paste into configs"
	@echo "tag          - print the tag used for this container"
	@echo "shell        - get a shell into the running container"

build:
	cd docker/ && docker build -t $(BIND_TAG) -f ./Dockerfile.master .

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

shell:
	docker exec -it $(BIND_TAG) bash

tag:
	@echo $(BIND_TAG)
