start_stack:
	docker-compose up -d

circleintegration:
	docker build -t circletemp -f ./IntegrationDockerfile .	
	docker-compose up -d
	docker run --network chapter11servicessearch_default -w /go/src/github.com/building-microservices-with-go/chapter11-services-search/features -e "MYSQL_CONNECTION=root:password@tcp(mysql:3306)/kittens" circletemp godog ./
	docker-compose stop
	docker-compose rm -f

integration: start_stack
	cd features && MYSQL_CONNECTION="root:password@tcp(${DOCKER_IP}:3306)/kittens" godog ./
	docker-compose stop
	docker-compose rm -f

unit:
	go test -v --race $(shell go list ./... | grep -v /vendor/)

benchmark:
	go test -bench=. github.com/building-microservices-with-go/chapter11-services-search/handlers

build_linux:
	CGO_ENABLED=0 GOOS=linux go build -o ./search .

build_docker:
	docker build -t buildingmicroserviceswithgo/search .

run: start_stack
	go run main.go
	docker-compose stop

test: unit benchmark integration
