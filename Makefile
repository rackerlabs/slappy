DOCKER_TAG := slappy-build

help:
	@echo ""
	@echo "build          - builds a slappy executable"
	@echo "run            - runs slappy"
	@echo "test           - runs slappy tests"
	@echo "clean          - cleans up built binaries"
	@echo ""
	@echo "dependencies   - go gets all the dependencies"
	@echo ""
	@echo "docker-build   - builds slappy via docker"
	@echo ""

build: fmt
	go build -o slappy main.go

run:
	./slappy -debug

fmt:
	find . -name '*.go' -exec go fmt '{}' \;

dependencies:
	go get github.com/rackerlabs/dns
	go get github.com/rackerlabs/iniflags

test:
	./slappy -debug &
	.venv/bin/python send14.py
	.venv/bin/python sendnotify.py
	pkill slappy

docker-build:
	docker build -t $(DOCKER_TAG) .
	docker run -v `pwd`:/build $(DOCKER_TAG) cp slappy /build

clean:
	rm -rf slappy
