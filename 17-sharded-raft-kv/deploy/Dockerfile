FROM golang:1.25.0-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY 17-sharded-raft-kv/*.go ./17-sharded-raft-kv/
COPY 17-sharded-raft-kv/api ./17-sharded-raft-kv/api
RUN CGO_ENABLED=0 go build -trimpath -o /raft ./17-sharded-raft-kv

FROM scratch
COPY --from=build /raft /raft
USER 10001:10001
EXPOSE 8091 8084
ENTRYPOINT ["/raft"]
