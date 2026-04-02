FROM docker.io/golang:1.26.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /out/api ./cmd/api && \
	go build -o /out/worker ./cmd/worker

FROM docker.io/alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/worker /usr/local/bin/worker
COPY --from=builder /src/docs ./docs
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/seeds ./seeds

EXPOSE 8080

CMD ["/usr/local/bin/api"]