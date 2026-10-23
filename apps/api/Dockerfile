# syntax=docker/dockerfile:1.10
# One build, two images: `--target imgworker` for the image worker, the default target for the API.
# ---- build ----
FROM golang:1.27-alpine AS build
WORKDIR /src
# nodynamic: the image codecs never try to load native libraries (keeps the binaries static
# for distroless). wasm2go: the HEIC decoder is compiled Go, with no runtime code generation.
ENV CGO_ENABLED=0 GOFLAGS="-trimpath -tags=nodynamic,wasm2go"
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download && go mod verify
COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o /out/api ./cmd/api && \
    go build -ldflags="-s -w" -o /out/imgworker ./cmd/imgworker

# ---- image worker: distroless, no shell, non-root ----
FROM gcr.io/distroless/static-debian12:nonroot AS imgworker
COPY --from=build /out/imgworker /imgworker
USER nonroot:nonroot
EXPOSE 9090
ENV WORKER_ADDR=:9090
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/imgworker", "healthcheck"]
ENTRYPOINT ["/imgworker"]

# ---- API: distroless, no shell, non-root ----
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENV HTTP_ADDR=:8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
