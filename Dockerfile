FROM golang:1.5

# Create an app dir, and copy our repo in
# Also a /build dir that will be a volume to . so we can copy
# our binary out with a:
# docker run -v `pwd`:/build $(DOCKER_TAG) cp slappy /build
RUN mkdir -p /go/src/github.com/rackerlabs/slappy
RUN mkdir /build
ADD . /go/src/github.com/rackerlabs/slappy/
WORKDIR /go/src/github.com/rackerlabs/slappy

RUN make build
