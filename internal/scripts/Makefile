.PHONY:	build push publish

PREFIX = registry.connect.redhat.com/coralogix
IMAGE = coralogix-fluentd
TAG ?= $(or ${VERSION},1.0.0)

build:
	docker build \
		--tag $(PREFIX)/$(IMAGE):$(TAG) \
		--file ../build/Dockerfile \
		--build-arg VERSION=$(TAG) \
		..

push:
	docker push $(PREFIX)/$(IMAGE):$(TAG)

publish: build push