.PHONY: test
test: FLAGS ?= -race -shuffle=on
test: PACKAGE ?= ./...
test:
	go test $(FLAGS) $(PACKAGE)


