FROM golang:1.17.0 as go-builder

WORKDIR /milesChallenge
COPY ./ /milesChallenge/
RUN go get ./...
RUN go build .

FROM ubuntu:focal
COPY --from=go-builder /milesChallenge/miles-challenge /usr/local/bin
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

EXPOSE 8081
