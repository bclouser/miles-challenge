FROM golang:1.17.0 as go-builder

WORKDIR /milesChallenge
COPY ./ /milesChallenge/
RUN go get ./...
RUN go build .

FROM ubuntu:focal
COPY --from=go-builder /milesChallenge/miles-challenge /usr/local/bin
RUN ln -fs /usr/share/zoneinfo/America/New_York /etc/localtime
RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y  tzdata ca-certificates && update-ca-certificates

EXPOSE 8081
