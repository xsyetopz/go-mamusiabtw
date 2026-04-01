FROM golang:1.26.1-bookworm AS builder

WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libsqlite3-dev \
  && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w" -o /out/jagpda ./cmd/jagpda


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
  && rm -rf /var/lib/apt/lists/*

ARG UID=1000
ARG GID=1000
RUN groupadd -g "${GID}" jagpda && useradd -u "${UID}" -g "${GID}" -m -s /usr/sbin/nologin jagpda

WORKDIR /app

COPY --from=builder /out/jagpda /usr/local/bin/jagpda
COPY migrations ./migrations
COPY locales ./locales
COPY plugins ./plugins
COPY config ./config

RUN mkdir -p /data && chown -R jagpda:jagpda /data

USER jagpda:jagpda

ENV SQLITE_PATH=/data/jagpda.sqlite
ENV MIGRATIONS_DIR=/app/migrations/sqlite
ENV LOCALES_DIR=/app/locales
ENV PLUGINS_DIR=/app/plugins
ENV JAGPDA_PERMISSIONS_FILE=/app/config/permissions.json

ENTRYPOINT ["jagpda"]
