FROM golang:onbuild

RUN mkdir /app
RUN mkdir /build

ADD . /app/

WORKDIR /app

RUN go build -o slappy .
