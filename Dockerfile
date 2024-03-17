FROM golang:1.21 AS builder
ENV CGO_ENABLED=0

WORKDIR /src
COPY . /src

RUN go mod tidy
RUN go build -o "/src/bin/go-archiver"

# We don't need golang to run binaries, just use alpine.
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /src/bin/go-archiver /app/go-archiver
COPY fourq_verify /app/fourq_verify
RUN chmod +x /app/go-archiver
RUN chmod +x /app/fourq_verify

EXPOSE 8000
EXPOSE 8001

WORKDIR /app

ENTRYPOINT ["./go-archiver"]
