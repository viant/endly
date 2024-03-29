FROM golang:1.11.5-alpine3.8

RUN mkdir -p /app
WORKDIR /app
COPY main.go /app
COPY index.html /app

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -o helloworld