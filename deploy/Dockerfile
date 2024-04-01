FROM golang:alpine

RUN apk update && apk upgrade && apk add --no-cache zfs

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd 
COPY internal ./internal 

RUN CGO_ENABLED=0 GOOS=linux go build -o /unraid-simple-monitoring-api ./cmd

COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]

ARG HOSTFS_PREFIX

ENV CONF_PATH=/app/conf.yml
ENV HOSTFS_PREFIX=${HOSTFS_PREFIX}

CMD ["/unraid-simple-monitoring-api"]
