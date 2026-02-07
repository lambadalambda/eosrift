ARG GO_VERSION=1.23
ARG VERSION=dev

FROM golang:${GO_VERSION} AS build
WORKDIR /src
ARG VERSION

COPY go.mod go.sum ./

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X eosrift.com/eosrift/internal/cli.version=${VERSION}" -o /out/eosrift-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w -X eosrift.com/eosrift/internal/cli.version=${VERSION}" -o /out/eosrift ./cmd/client
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/eosrift-deployhook ./cmd/deployhook

FROM gcr.io/distroless/base-debian12 AS server
COPY --from=build /out/eosrift-server /eosrift-server
EXPOSE 8080
ENV EOSRIFT_LISTEN_ADDR=:8080
ENTRYPOINT ["/eosrift-server"]

FROM gcr.io/distroless/base-debian12 AS client
COPY --from=build /out/eosrift /eosrift
ENTRYPOINT ["/eosrift"]

FROM docker:27-cli AS deployhook
RUN apk add --no-cache curl
COPY --from=build /out/eosrift-deployhook /usr/local/bin/eosrift-deployhook
ENTRYPOINT ["/usr/local/bin/eosrift-deployhook"]
