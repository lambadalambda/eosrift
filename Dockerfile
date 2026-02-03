ARG GO_VERSION=1.23

FROM golang:${GO_VERSION} AS build
WORKDIR /src

COPY go.mod go.sum ./

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/eosrift-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/eosrift ./cmd/client

FROM gcr.io/distroless/base-debian12 AS server
COPY --from=build /out/eosrift-server /eosrift-server
EXPOSE 8080
ENV EOSRIFT_LISTEN_ADDR=:8080
ENTRYPOINT ["/eosrift-server"]

FROM gcr.io/distroless/base-debian12 AS client
COPY --from=build /out/eosrift /eosrift
ENTRYPOINT ["/eosrift"]
