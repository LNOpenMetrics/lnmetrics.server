CC=go
FMT=gofmt
NAME=lnmetricsd
BASE_DIR=/script
OS=linux
ARCH=386

default: fmt lint build

build:
	$(CC) build -o $(NAME) cmd/lnmetrics.server/main.go

fmt:
	$(CC) fmt ./...

check:
	echo "Nothings yet"

lint:
	golangci-lint run

gen:
	$(CC) run github.com/99designs/gqlgen generate

release:
	env GOOS=$(OS) GOARCH=$(ARCH) $(CC) build -o $(NAME)-$(OS)-$(ARCH) cmd/lnmetrics.server/main.go
