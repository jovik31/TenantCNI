kind-cluster:
	kind create cluster --name=cluster --config=manifests/cluster_kind_deployment.yaml

kind-delete:
	kind delete cluster --name=cluster

create-image:
	docker build -t jovik31/tenantcni:0.2.1 .


kind-load-image:
	kind load docker-image jovik31/tenantcni:0.2.1 --name=cluster

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

