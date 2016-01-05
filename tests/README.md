slappy tests
------------

These are integration tests for slappy. Master and target bind servers are
provided to slappy using docker containers.


### How to

Since these are functional tests, we need actual bind nameservers to test
against. This is accomplished with two docker containers. One container is
the master (running just bind), and the other container is the target (running
both slappy and bind).

You'll need to:

1. Install docker
2. Install go
3. Build slappy
4. Install test dependencies
5. Build the docker containers
6. Run the tests

There are makefiles to help with some of lot of this.


##### Build slappy

Install go and build slappy:

1. Install go.
2. Set your `$GOPATH` to some directory and `go get github.com/rackerlabs/slappy`.
3. `cd` into `$GOPATH/src/github.com/rackerlabs/slappy`
4. Build slappy with a `make build`. This should put `slappy` at `$GOPATH/bin`.

The makefiles will search `$GOPATH/bin` for the slappy executable to copy into
the docker container.


##### Build the docker images

Now build and start the docker containers using a makefile:

1. Move into the tests directory: `cd tests`
2. `make start-containers` to build the images and start the containers
3. `make check-containers` will fail if either bind or slappy are not running


##### Running the tests

Install some dependencies and run the tests

1. `make write-test-config` will write out a `test.conf` file for the tests
2. `pip install tox` - tox is used to run the tests
3. `tox -e py27` will run the tests
