.PHONY: run test cover build docker-build k8s-deploy k8s-remove

run:
	go run ./cmd/server

test:
	go test ./... -race -cover

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

build:
	go build -o bin/server ./cmd/server

docker-build:
	docker build -t math-api:local .

k8s-deploy:
	kubectl apply -k deploy/k8s

k8s-remove:
	kubectl delete -k deploy/k8s
