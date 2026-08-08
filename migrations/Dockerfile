FROM golang:1.26-alpine AS build

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

RUN CGO_ENABLED=0 GOBIN=/out go install -tags postgres \
    github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

FROM alpine:3.24.1

COPY --from=build /out/migrate /usr/local/bin/migrate
COPY . /migrations

ENTRYPOINT ["migrate"]
