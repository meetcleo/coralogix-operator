.PHONY:	build push clean publish render deploy

PREFIX = registry.connect.redhat.com/coralogix
IMAGE = coralogix-operator
TAG ?= $(or ${VERSION},1.0.0)

clean:
	@rm -rf build/_output

render:
	@rm -f coralogix-operator.yaml
	@kubectl kustomize deploy > coralogix-operator.yaml

build: clean
	operator-sdk build $(PREFIX)/$(IMAGE):$(TAG)
	@sed -i 's|$(PREFIX)/$(IMAGE):.*|$(PREFIX)/$(IMAGE):$(TAG)|g' deploy/operator.yaml

push:
	@docker push $(PREFIX)/$(IMAGE):$(TAG)

deploy: render
	@kubectl create -f deploy/crds/coralogix.com_coralogixloggers_crd.yaml
	@kubectl create -f coralogix-operator.yaml

undeploy:
	@kubectl delete -f deploy/crds/coralogix.com_coralogixloggers_crd.yaml
	@kubectl delete -f coralogix-operator.yaml

package:
	@cp -f deploy/crds/coralogix.com_coralogixloggers_crd.yaml bundle/coralogix-operator.v$(TAG).crd.yaml
	@cp -f deploy/olm-catalog/coralogix-operator/$(TAG)/coralogix-operator.clusterserviceversion.yaml bundle/coralogix-operator.v$(TAG).clusterserviceversion.yaml

publish: build render push