DOCKER_TAG := slappy-build
TEST_VENV := tests/.venv

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
		test -d $(TEST_VENV) || virtualenv $(TEST_VENV)
		$(TEST_VENV)/bin/pip install -r tests/test-requirements.txt
		make -C tests start-containers
		make -C tests check-containers
		make -C tests write-test-config
		$(TEST_VENV)/bin/tox -c tests/tox.ini -e py27
		make -C tests stop-containers

docker-build:
		docker build -t $(DOCKER_TAG) .
		docker run -v `pwd`:/build $(DOCKER_TAG) cp slappy /build
		@echo "If you're using docker-machine, run:"
		@echo 'docker-machine scp $$DOCKER_MACHINE_NAME:$$(pwd)/slappy .'

clean:
		rm -rf slappy

dependencies:
		git submodule update --init
