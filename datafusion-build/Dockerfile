FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir \
    flask \
    flask-cors \
    datafusion \
    pyiceberg \
    boto3 \
    pyarrow

COPY datafusion_server.py /app/server.py

# REST SQL endpoint on 8555 (canonical Uisce port)
ENV FLIGHT_SQL_PORT=8555
ENV RUST_LOG=info

EXPOSE 8554 8555

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=5 \
    CMD curl -f http://localhost:8555/health || exit 1

ENTRYPOINT ["python", "/app/server.py"]
