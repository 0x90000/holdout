FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /holdout ./cmd/holdout

FROM alpine:3.20
RUN apk add --no-cache python3
COPY --from=build /holdout /usr/local/bin/holdout
COPY tasks /opt/holdout/tasks
COPY schema /opt/holdout/schema
WORKDIR /opt/holdout
ENTRYPOINT ["/usr/local/bin/holdout"]
