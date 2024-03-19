kind-cluster:
	kind create cluster --name=kind-cluster --config=manifests/cluster_kind_deployment.yaml

kind-delete:
	kind delete cluster --name=kind-cluster

create-image:
	docker build -t jovik31/tenant:0.1.0 .


kind-load-image:
	kind load docker-image jovik31/tenant:0.1.0 --name=kind-cluster

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build