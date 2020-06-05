FROM golang:1.14 AS build

COPY . /src/project
WORKDIR /src/project

RUN export CGO_ENABLED=0 &&\
    export GOPROXY=https://goproxy.io &&\
    make &&\
    cp cmd/alarm-adapter/alarm-adapter /alarm-adapter &&\
    cp etc/alarm-adapter.sample.conf /alarm-adapter.conf

FROM debian:10
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=build /alarm-adapter /alarm-adapter
COPY --from=build /alarm-adapter.conf /etc/alarm-adapter.conf

CMD ["/alarm-adapter", "start", "-c", "/etc/alarm-adapter.conf"]
