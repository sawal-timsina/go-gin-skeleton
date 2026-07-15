# syntax=docker/dockerfile:1
#
# Production image. Multi-stage: a full toolchain builds the binary, a slim
# non-root Alpine image runs it. Distinct from docker/web.Dockerfile, which is
# the hot-reload development image.

# ---- build stage ----
FROM golang:1.23-alpine AS build

# webp is a cgo package, so a C toolchain is required to build; git lets the Go
# toolchain stamp VCS info.
RUN apk add --no-cache build-base git

WORKDIR /src

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/app .

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build /out/app /app/app
# Runtime assets read from disk at startup.
COPY --from=build /src/database/migration /app/database/migration
COPY --from=build /src/templates /app/templates

USER app

# HTTP (also serves /metrics, /livez, /readyz) and gRPC.
EXPOSE 8080 9090

ENTRYPOINT ["/app/app"]
