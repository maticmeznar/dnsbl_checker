# NAME is set to the name of the current directory ($PWD)
NAME := $(notdir $(shell pwd))
DISTDIR := ./dist

default: build

build:
	go build

buildall: buildwin buildlinux builddarwin

buildwin:
	env GOOS=windows GOARCH=amd64 go build -o ${DISTDIR}/${NAME}_windows_amd64.exe
	zip -j -9 ${DISTDIR}/${NAME}_windows_amd64.zip ${DISTDIR}/${NAME}_windows_amd64.exe

	env GOOS=windows GOARCH=386 go build -o ${DISTDIR}/${NAME}_windows_386.exe
	zip -j -9 ${DISTDIR}/${NAME}_windows_386.zip ${DISTDIR}/${NAME}_windows_386.exe

buildlinux:
	env GOOS=linux GOARCH=386 go build -o ${DISTDIR}/${NAME}_linux_386
	zip -j -9 ${DISTDIR}/${NAME}_linux_386.zip ${DISTDIR}/${NAME}_linux_386

	env GOOS=linux GOARCH=amd64 go build -o ${DISTDIR}/${NAME}_linux_amd64
	zip -j -9 ${DISTDIR}/${NAME}_linux_amd64.zip ${DISTDIR}/${NAME}_linux_amd64

builddarwin:
	env GOOS=darwin GOARCH=amd64 go build -o ${DISTDIR}/${NAME}_darwin_amd64
	zip -j -9 ${DISTDIR}/${NAME}_darwin_amd64.zip ${DISTDIR}/${NAME}_darwin_amd64

brun: build
	./${NAME}

clean:
	rm -f ./${NAME}
	rm ${DISTDIR}/*

test:
	@echo ${NAME}
