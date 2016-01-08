DOCKER_TAG := slappy-build

help:
		@echo ""
		@echo "build          - builds a slappy executable"
		@echo "run            - runs slappy"
		@echo "test           - runs slappy tests"
		@echo "clean          - cleans up built binaries"
		@echo ""
		@echo "dependencies   - gets all the dependencies pulled to the submodules in vendor/"
		@echo ""
		@echo "docker-build   - builds slappy via docker"
		@echo ""

build: fmt
		GO15VENDOREXPERIMENT=1 go build -o slappy -ldflags "-X main.builddate=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.gitref=`git rev-parse HEAD`" main.go

run:
		./slappy -debug

fmt:
		find . -maxdepth 2 -name '*.go' -exec go fmt '{}' \;

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

dependencies:
		git submodule update --init
