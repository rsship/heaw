build: 
	@go build -o bin/ . 

run: build
	@bin/heaw

default: run

.PHONY: build run

