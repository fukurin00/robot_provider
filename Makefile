# Makefile for Geo Routing Provider

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
RM=rm

LDFLAGS="-X github.com/synerex/synerex_sxutil.GitVer=`git describe --tag` -X github.com/synerex/synerex_sxutil.buildTime=`date +%Y-%m-%d_%T` -X github.com/synerex/synerex_sxutil.Sha1Ver=`git rev-parse HEAD`"

TARGET=robot-provider
# Main target

.PHONY: build 
build: $(TARGET)

robot-provider: main.go
	$(GOBUILD) -o $(TARGET) main.go 

.PHONY: clean
clean: 
	$(RM) $(TARGET)
