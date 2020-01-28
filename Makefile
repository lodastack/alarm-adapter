all: build

fmt:
	gofmt -l -w -s ./

dep:fmt

install:dep

build:dep
	cd cmd/alarm-adapter && go build -v -mod=vendor

clean:
	cd cmd/alarm-adapter && go clean
