all: build

fmt:
	gofmt -l -w -s ./

dep:fmt
	go get github.com/sparrc/gdm
	go install github.com/sparrc/gdm
	/go/bin/gdm restore

install:dep
	go install agent

build:dep
	cd cmd/alarm-adapter && go build -v

clean:
	cd cmd/alarm-adapter && go clean
