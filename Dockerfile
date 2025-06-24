FROM golang:1.24 AS builder
ENV CGO_ENABLED=0

WORKDIR /src
COPY . /src

RUN go mod tidy
RUN go build -o "/src/bin/go-archiver"

# We don't need golang to run binaries, just use alpine.
FROM ubuntu:24.04
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /src/bin/go-archiver /app/go-archiver
RUN chmod +x /app/go-archiver

EXPOSE 8000
EXPOSE 8001
EXPOSE 8002

WORKDIR /app

ENTRYPOINT ["./go-archiver"]
