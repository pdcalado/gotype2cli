VERSION ?= $(shell git describe --abbrev=7 || echo -n "unversioned")
VERSION_PACKAGE ?= github.com/pdcalado/gotype2cli/version

LDFLAGS ?= "-X '$(VERSION_PACKAGE).Version=$(VERSION)' -s -w"

GOBUILD ?= GCO_ENABLED=0 go build -ldflags=$(LDFLAGS) -tags osusergo,netgo

fmt:
	gofmt -w -s ./
	goimports -w -local github.com/pdcalado/gotype2cli ./

lint:
	golangci-lint run -v

clean:
	rm -rf ./bin

build:
	$(GOBUILD) -o gotype2cli ./cmd

e2e:
	rm -f testdata/bar/bar_command.go
	go run ./cmd -type Bar ./testdata/bar/ > /tmp/bar_command.go
	cp /tmp/bar_command.go testdata/bar
	go build -o bar ./testdata/bar
	./bar -h
	./bar raise | grep "height\":1"
	./bar raise | ./bar raise | grep "height\":2"
	./bar raise | ./bar raise-by 3 | grep "height\":4"
	./bar raise-by 2 | ./bar string | grep "the bar is 2 meters high"
	./bar new | ./bar raise | grep "height\":13"
