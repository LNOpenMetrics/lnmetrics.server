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

check-full:
	richgo test ./tests -v
	richgo test ./... -v

check:
	$(CC) test ./...

lint:
	golangci-lint run

gen:
	$(CC) run ./scripts/gqlgen.go --verbose

mod:
	$(CC) get -d github.com/kyoh86/richgo
	$(CC) mod vendor

release:
	env GOOS=$(OS) GOARCH=$(ARCH) $(CC) build -o $(NAME)-$(OS)-$(ARCH) cmd/lnmetrics.server/main.go

prod:
	$(CC) build -ldflags '-s -w' -o $(NAME) cmd/lnmetrics.server/main.go

dep:
	$(CC) get -d github.com/LNOpenMetrics/lnmetrics.utils
	$(CC) mod vendor

trace:
	GOTRACEBACK=system $(CC) run cmd/lnmetrics.server/main.go

