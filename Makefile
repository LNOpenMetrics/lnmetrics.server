CC=go
FMT=gofmt
NAME=lnmetricsd
BASE_DIR=/script
OS=linux
ARCH=386

default: fmt gen
	$(CC) build -o $(NAME) cmd/ln-metrics-server/server.go

fmt:
	$(CC) fmt ./...

check:
	echo "Nothings yet"

gen:
	$(CC) run github.com/99designs/gqlgen generate

build:
	env GOOS=$(OS) GOARCH=$(ARCH) $(CC) build -o $(NAME)-$(OS)-$(ARCH) cmd/ln-metrics-server/server.go
