FROM golang:1.25-alpine AS build
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/samkc-0/pamphlet-sync/internal/version.CommitSHA=${COMMIT_SHA} -X github.com/samkc-0/pamphlet-sync/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /usr/local/bin/api

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
