all: build

fmt:
	gofmt -l -w -s ./

dep:fmt
	gdm restore

install:dep
	go install agent

build:dep
	cd cmd/alarm-adapter && go build -v

clean:
	cd cmd/alarm-adapter && go clean
