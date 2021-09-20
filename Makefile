GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install

build:
	$(GOBUILD) -o ./cmd/udp-server/udp-server ./cmd/udp-server/

install:
	$(GOINSTALL) ./...

run-server:
	$(GOBUILD) -o ./cmd/udp-server/udp-server ./cmd/udp-server/
	./cmd/udp-server/udp-server