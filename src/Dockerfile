FROM golang:1.17.0 as go-builder

WORKDIR /milesChallenge
COPY ./ /milesChallenge/
RUN go get ./...
RUN go build . -o miles-challenge

FROM ubuntu:focal
COPY --from=go-builder /milesChallenge/miles-challenge /usr/local/bin
