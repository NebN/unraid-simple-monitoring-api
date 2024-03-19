FROM golang:alpine

RUN apk update && apk upgrade

RUN mkdir -p /app/cmd

WORKDIR /app

COPY ../cmd cmd 
COPY ../go.mod .

CMD ["go", "run", "/app/cmd/main.go"]

