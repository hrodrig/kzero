# kzero — minimal runtime image (build from repo root: make docker-build)
# Final stage uses distroless (no Alpine/BusyBox) so CVEs in wget/busybox from
# minimal Alpine bases do not apply; CA certs are included in distroless static.
FROM golang:1.26.5-alpine3.24 AS build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDDATE=unknown
ARG BRANCH=unknown
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY configs/ ./configs/
RUN CGO_ENABLED=0 go build -trimpath \
	-ldflags "-s -w -X github.com/hrodrig/kzero/internal/cli.Version=${VERSION} -X github.com/hrodrig/kzero/internal/notify.AppVersion=${VERSION} -X github.com/hrodrig/kzero/internal/cli.Commit=${COMMIT} -X github.com/hrodrig/kzero/internal/cli.BuildDate=${BUILDDATE} -X github.com/hrodrig/kzero/internal/cli.Branch=${BRANCH}" \
	-o /kzero ./cmd/kzero

FROM gcr.io/distroless/static-debian13:nonroot
LABEL org.opencontainers.image.title="kzero"
LABEL org.opencontainers.image.description="Declarative Kubernetes workload pipelines"
LABEL org.opencontainers.image.source="https://github.com/hrodrig/kzero"
COPY --from=build /kzero /usr/local/bin/kzero
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/kzero"]
