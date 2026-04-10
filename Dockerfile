FROM golang:1.26.2-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ARG BUILD_VERSION=dev
ARG BUILD_REPOSITORY=https://github.com/xsyetopz/go-mamusiabtw
ARG BUILD_DESCRIPTION="A nurturing and protective Discord app."
ARG BUILD_DEVELOPER_URL=
ARG BUILD_SUPPORT_SERVER_URL=
ARG BUILD_MASCOT_IMAGE_URL=
RUN go build -trimpath \
  -ldflags="-s -w \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.Version=${BUILD_VERSION}' \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.Repository=${BUILD_REPOSITORY}' \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.Description=${BUILD_DESCRIPTION}' \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.DeveloperURL=${BUILD_DEVELOPER_URL}' \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.SupportServerURL=${BUILD_SUPPORT_SERVER_URL}' \
    -X 'github.com/xsyetopz/go-mamusiabtw/internal/buildinfo.MascotImageURL=${BUILD_MASCOT_IMAGE_URL}'" \
  -o /out/mamusiabtw ./cmd/mamusiabtw


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
  && rm -rf /var/lib/apt/lists/*

ARG UID=1000
ARG GID=1000
RUN groupadd -g "${GID}" mamusiabtw && useradd -u "${UID}" -g "${GID}" -m -s /usr/sbin/nologin mamusiabtw

WORKDIR /app

COPY --from=builder /out/mamusiabtw /usr/local/bin/mamusiabtw
COPY migrations ./migrations
COPY locales ./locales
COPY plugins ./plugins
COPY config ./config

RUN mkdir -p /data && chown -R mamusiabtw:mamusiabtw /data

USER mamusiabtw:mamusiabtw

ENV SQLITE_PATH=/data/mamusiabtw.sqlite
ENV MIGRATIONS_DIR=/app/migrations/sqlite
ENV LOCALES_DIR=/app/locales
ENV PLUGINS_DIR=/app/plugins
ENV MAMUSIABTW_PERMISSIONS_FILE=/app/config/permissions.json

ENTRYPOINT ["mamusiabtw"]
