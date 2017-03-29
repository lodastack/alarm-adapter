FROM ubuntu:latest

WORKDIR /app

# copy binary and config into image
COPY alarm-adapter /app/
COPY alarm-adapter.conf /app/

RUN chmod +x alarm-adapter

# Add influxd to the PATH
ENV PATH=/app:$PATH

VOLUME ["/data/logs"]

ENTRYPOINT ["alarm-adapter", "start", "-c", "alarm-adapter.conf"]