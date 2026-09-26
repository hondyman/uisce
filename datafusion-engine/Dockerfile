# DataFusion engine: file profile/read/write endpoints for the data pipeline.
FROM rust:1.89-bookworm AS build
WORKDIR /src
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release --locked

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* \
 && useradd --system --uid 10001 --home /nonexistent engine \
 && mkdir -p /data/files && chown engine /data/files
COPY --from=build /src/target/release/datafusion-engine /usr/local/bin/datafusion-engine
USER engine
ENV PORT=8091 DATAFUSION_FILE_ROOT=/data/files
EXPOSE 8091
HEALTHCHECK --interval=30s --timeout=3s CMD curl -fsS http://localhost:8091/health || exit 1
ENTRYPOINT ["/usr/local/bin/datafusion-engine"]
