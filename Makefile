PROJECT = micro-srv-test
REGISTRY ?= slarkin
IMAGE := $(REGISTRY)/$(PROJECT)

GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)
VERSION ?= $(GIT_REF)

export GO111MODULE=on

api:
	protoc --go_out=plugins=grpc:greet/ -I greet/ greet/*.proto

container: api
	docker build . -t $(IMAGE):$(VERSION)

push: container
	docker push $(IMAGE):$(VERSION)
