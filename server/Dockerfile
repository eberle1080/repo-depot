FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Copy workspace and module files first for layer caching
COPY go.work go.work.sum* ./
COPY shared/go.mod shared/go.sum* ./shared/
COPY server/go.mod server/go.sum* ./server/

RUN go work sync && \
    cd server && go mod download

# Copy source
COPY shared/ ./shared/
COPY server/ ./server/

WORKDIR /build/server
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/repo-depot-server .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/repo-depot-server /usr/local/bin/repo-depot-server

EXPOSE 50051

ENTRYPOINT ["repo-depot-server"]
