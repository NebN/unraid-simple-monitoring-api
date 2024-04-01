FROM golang:alpine as build

RUN apk update && apk upgrade

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd 
COPY internal ./internal 

RUN CGO_ENABLED=0 GOOS=linux go build -o /unraid-simple-monitoring-api ./cmd


FROM alpine as run

RUN apk update && apk upgrade && apk add --no-cache zfs

COPY --from=build /unraid-simple-monitoring-api .

COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ARG HOSTFS_PREFIX

ENV CONF_PATH=/app/conf.yml
ENV HOSTFS_PREFIX=${HOSTFS_PREFIX}

ENTRYPOINT [ "/entrypoint.sh" ]

CMD ["/unraid-simple-monitoring-api"]
