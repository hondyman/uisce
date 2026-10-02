# MinIO server and mc client, built from source at a pinned release tag.
#
# MinIO stopped publishing container images (minio/minio and minio/mc are gone
# from Docker Hub, quay.io has no matching tags) and removed its release
# binaries from dl.min.io (HTTP 410), but the source tags are still public. The
# compose files build this image themselves, so there is no registry to host or
# authenticate to. Bump the tags deliberately.
#
#   docker build --target minio -t uisce/minio docker/minio
#   docker build --target mc    -t uisce/mc    docker/minio

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS build-minio
ARG MINIO_TAG=RELEASE.2025-10-15T17-29-55Z
RUN apk add --no-cache git
RUN git clone --depth 1 --branch "${MINIO_TAG}" https://github.com/minio/minio.git /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -trimpath -tags kqueue -ldflags "$(go run buildscripts/gen-ldflags.go)" -o /out/minio .

FROM golang:${GO_VERSION}-alpine AS build-mc
ARG MC_TAG=RELEASE.2025-08-13T08-35-41Z
RUN apk add --no-cache git
RUN git clone --depth 1 --branch "${MC_TAG}" https://github.com/minio/mc.git /src
WORKDIR /src
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-s -w" -o /out/mc .

FROM alpine:3.20 AS minio
ARG MINIO_TAG=RELEASE.2025-10-15T17-29-55Z
LABEL name="MinIO" version="${MINIO_TAG}" \
      org.opencontainers.image.source="https://github.com/minio/minio" \
      org.opencontainers.image.licenses="AGPL-3.0-only"
# curl: the compose healthchecks call it inside the container.
RUN apk add --no-cache ca-certificates curl
ENV MINIO_ACCESS_KEY_FILE=access_key \
    MINIO_SECRET_KEY_FILE=secret_key \
    MINIO_ROOT_USER_FILE=access_key \
    MINIO_ROOT_PASSWORD_FILE=secret_key \
    MINIO_KMS_SECRET_KEY_FILE=kms_master_key \
    MINIO_CONFIG_ENV_FILE=config.env \
    MC_CONFIG_DIR=/tmp/.mc
COPY --from=build-minio /out/minio /usr/bin/minio
COPY --from=build-minio /src/dockerscripts/docker-entrypoint.sh /usr/bin/docker-entrypoint.sh
RUN chmod +x /usr/bin/minio /usr/bin/docker-entrypoint.sh
EXPOSE 9000
VOLUME ["/data"]
ENTRYPOINT ["/usr/bin/docker-entrypoint.sh"]
CMD ["minio"]

FROM alpine:3.20 AS mc
ARG MC_TAG=RELEASE.2025-08-13T08-35-41Z
LABEL name="MinIO Client" version="${MC_TAG}" \
      org.opencontainers.image.source="https://github.com/minio/mc" \
      org.opencontainers.image.licenses="AGPL-3.0-only"
RUN apk add --no-cache ca-certificates
ENV MC_CONFIG_DIR=/tmp/.mc
COPY --from=build-mc /out/mc /usr/bin/mc
RUN chmod +x /usr/bin/mc
ENTRYPOINT ["/usr/bin/mc"]
