# NAME is set to the name of the current directory ($PWD)
NAME := $(notdir $(shell pwd))

default: build

build:
	go build

buildall: buildwin buildlinux buildmacos

buildwin:
	env GOOS=windows GOARCH=amd64 go build -o ${NAME}_windows_amd64.exe
	env GOOS=windows GOARCH=386 go build -o ${NAME}_windows_386.exe

buildlinux:
	env GOOS=linux GOARCH=386 go build -o ${NAME}_linux_386
	env GOOS=linux GOARCH=amd64 go build -o ${NAME}_linux_amd64

buildmacos:
	env GOOS=darwin GOARCH=amd64 go build -o ${NAME}_darwin_amd64

brun: build
	./${NAME}

test:
	@echo ${NAME}
