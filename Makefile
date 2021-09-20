GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install

build:
	$(GOBUILD) -o ./cmd/udp-server/udp-server ./cmd/udp-server/
	$(GOBUILD) -o ./cmd/udp-client/udp-client ./cmd/udp-client/

install:
	$(GOINSTALL) ./...

run-server:
	$(GOBUILD) -o ./cmd/udp-server/udp-server ./cmd/udp-server/
	./cmd/udp-server/udp-server

run-client:
	$(GOBUILD) -o ./cmd/udp-client/udp-client ./cmd/udp-client/
	./cmd/udp-client/udp-client