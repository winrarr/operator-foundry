# Build the manager binary.
ARG BASE_IMAGE=docker.io/library/golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea
ARG RUNTIME_IMAGE=gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3
FROM ${BASE_IMAGE} AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -ldflags="-s -w" -o manager ./cmd
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -ldflags="-s -w" -o mock-external-api ./cmd/mockexternalapi

ARG RUNTIME_IMAGE
FROM ${RUNTIME_IMAGE} AS mock-api-runtime
WORKDIR /
COPY --from=builder /workspace/mock-external-api .
USER 65532:65532
ENTRYPOINT ["/mock-external-api"]

FROM ${RUNTIME_IMAGE} AS manager-runtime
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
