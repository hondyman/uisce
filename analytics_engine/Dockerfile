FROM python:3.10-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Generate protobufs
RUN python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. analytics.proto

CMD ["python", "main.py"]
