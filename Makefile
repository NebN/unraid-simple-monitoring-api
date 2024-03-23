run: 
	export CONF_PATH="conf/conf.yml" && \
	go run ./cmd/main.go
test:
	go test ./...
docker-run:
	sudo docker compose --env-file .env/dev.env --file deploy/docker-compose.yml up --build --force-recreate -d 
docker-build:
	sudo docker compose --env-file .env/dev.env --file deploy/docker-compose.yml build 
docker-logs:
	sudo docker logs -f $(shell sudo docker ps | grep unraid-api | cut -d " " -f 1)
docker-push: test
	sudo docker compose --env-file .env/prod.env --file deploy/docker-compose.yml build --no-cache --push
docker-prune:
	sudo docker image prune -f	
