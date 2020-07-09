.PHONY:	build push publish

PREFIX = docker.io/coralogixrepo
IMAGE = fluentd-coralogix-ubi
TAG ?= $(or ${VERSION},1.0.0)

build:
	docker build \
		--tag $(PREFIX)/$(IMAGE):latest \
		--tag $(PREFIX)/$(IMAGE):$(TAG) \
		--file ../build/Dockerfile \
		--build-arg VERSION=$(TAG) \
		..

push:
	docker push $(PREFIX)/$(IMAGE):latest
	docker push $(PREFIX)/$(IMAGE):$(TAG)

publish: build push