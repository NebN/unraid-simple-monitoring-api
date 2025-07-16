run: 
	export CONF_PATH="conf/conf.yml" && \
	go run ./cmd/main.go

test:
	go test -count=1 ./...

docker-run:
	sudo docker compose --env-file .env/dev.env --file deploy/docker-compose.yml up --build --force-recreate -d 

docker-build:
	sudo docker compose --env-file .env/dev.env --file deploy/docker-compose.yml build --no-cache

docker-rm:
	sudo docker compose --env-file .env/dev.env --file deploy/docker-compose.yml rm -f

docker-logs:
	sudo docker logs -f $(shell sudo docker ps | grep unraid-simple-monitoring-api | cut -d " " -f 1)

docker-push-qa: test
	sudo docker compose --env-file .env/prod.env --file deploy/docker-compose.yml build --no-cache && \
	sudo docker tag nebn/unraid-simple-monitoring-api:latest ghcr.io/nebn/unraid-simple-monitoring-api:qa && \
	sudo docker push ghcr.io/nebn/unraid-simple-monitoring-api:qa

docker-push-qa-internal: test
	sudo docker compose --env-file .env/prod.env --file deploy/docker-compose.yml build --no-cache && \
	sudo docker tag nebn/unraid-simple-monitoring-api:latest ghcr.io/nebn/unraid-simple-monitoring-api:qa-internal && \
	sudo docker push ghcr.io/nebn/unraid-simple-monitoring-api:qa-internal

docker-prune:
	sudo docker image prune -f && sudo docker container prune -f	
