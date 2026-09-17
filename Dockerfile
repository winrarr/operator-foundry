# Build the manager binary.
ARG BASE_IMAGE=golang:1.27
FROM ${BASE_IMAGE} AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -ldflags="-s -w" -o manager ./cmd
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -trimpath -ldflags="-s -w" -o mock-external-api ./cmd/mockexternalapi

FROM gcr.io/distroless/static:nonroot AS mock-api-runtime
WORKDIR /
COPY --from=builder /workspace/mock-external-api .
USER 65532:65532
ENTRYPOINT ["/mock-external-api"]

FROM gcr.io/distroless/static:nonroot AS manager-runtime
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
