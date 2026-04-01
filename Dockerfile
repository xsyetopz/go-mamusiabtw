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
RUN go build -trimpath -ldflags="-s -w" -o /out/imotherbtw ./cmd/imotherbtw


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
  && rm -rf /var/lib/apt/lists/*

ARG UID=1000
ARG GID=1000
RUN groupadd -g "${GID}" imotherbtw && useradd -u "${UID}" -g "${GID}" -m -s /usr/sbin/nologin imotherbtw

WORKDIR /app

COPY --from=builder /out/imotherbtw /usr/local/bin/imotherbtw
COPY migrations ./migrations
COPY locales ./locales
COPY plugins ./plugins
COPY config ./config

RUN mkdir -p /data && chown -R imotherbtw:imotherbtw /data

USER imotherbtw:imotherbtw

ENV SQLITE_PATH=/data/imotherbtw.sqlite
ENV MIGRATIONS_DIR=/app/migrations/sqlite
ENV LOCALES_DIR=/app/locales
ENV PLUGINS_DIR=/app/plugins
ENV IMOTHERBTW_PERMISSIONS_FILE=/app/config/permissions.json

ENTRYPOINT ["imotherbtw"]
