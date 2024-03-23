FROM golang:alpine

RUN apk update && apk upgrade

RUN mkdir -p /app/cmd

WORKDIR /app

COPY ../go.mod .
COPY ../go.sum .

RUN go mod download

COPY ../cmd cmd 

RUN export CONF_PATH=/app/conf.yml

CMD ["go", "run", "/app/cmd/main.go"]

