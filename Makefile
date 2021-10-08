# Set the shell to bash always
SHELL := /bin/bash

# Options
ORG_NAME=cr.yandex/crpq165pjbjiprv8pfri/crossplane
PROVIDER_NAME=provider-yc

build: generate test
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./bin/$(PROVIDER_NAME)-controller cmd/provider/main.go

image: generate test
	docker build . -t $(ORG_NAME)/$(PROVIDER_NAME):latest -f cluster/Dockerfile

image-tag:
	docker image tag sha256:$(SHA) $(ORG_NAME)/$(PROVIDER_NAME):$(VERSION)

image-push:
	docker push $(ORG_NAME)/$(PROVIDER_NAME):$(VERSION)

image-tag-n-push: image-tag image-push

run: generate
	kubectl apply -f package/crds/ -R
	go run cmd/provider/main.go -d

all: image image-push

generate:
	go generate ./...
	@find package/crds -name *.yaml -exec sed -i.sed -e '1,2d' {} \;
	@find package/crds -name *.yaml.sed -delete

lint:
	$(LINT) run

tidy:
	go mod tidy

test:
	go test -v ./...

# Tools

KIND=$(shell which kind)
LINT=$(shell which golangci-lint)

.PHONY: generate tidy lint clean build image all run
