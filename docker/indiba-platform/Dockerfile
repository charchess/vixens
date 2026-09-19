FROM python:3.13-alpine

RUN addgroup -g 10001 indiba && adduser -D -u 10001 -G indiba indiba \
    && apk add --no-cache ca-certificates

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY runtime.py .

USER 10001:10001
ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "/app/runtime.py"]
