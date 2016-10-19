PROGRAM = loglet

BUILD_NUMBER ?= SNAPSHOT-$(shell git rev-parse --short HEAD)

SOURCES := $(shell find . -name '*.go')

all: $(PROGRAM)

$(PROGRAM): $(SOURCES)
	CGO_ENABLED=0 go build -tags netgo -ldflags '-w' ./cmd/loglet 

clean:
	rm -rf $(PROGRAM)

.PHONY: clean

test:
	go test -v

test-cpu: $(PROGRAM)
	zcat journal.gz | ./loglet --log-level=debug --fake-elasticsearch --cpu-profile=loglet-cpu.prof

test-mem: $(PROGRAM)
	zcat journal.gz | ./loglet --log-level=debug --fake-elasticsearch --mem-profile=loglet-mem.prof
