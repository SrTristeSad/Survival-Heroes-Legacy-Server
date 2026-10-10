FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gateway ./cmd/gateway && \
    CGO_ENABLED=0 go build -o /out/game-server ./cmd/game-server && \
    CGO_ENABLED=0 go build -o /out/query-server ./cmd/query-server

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /out/gateway /usr/local/bin/gateway
COPY --from=build /out/game-server /usr/local/bin/game-server
COPY --from=build /out/query-server /usr/local/bin/query-server
ENTRYPOINT ["/usr/local/bin/gateway"]
