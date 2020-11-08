FROM golang:1.15 as builder
WORKDIR /workspace
COPY go.mod go.mod
RUN go mod download
COPY cmd/ cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o git-proxy cmd/git-proxy/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/ingress-healthz .
USER nonroot:nonroot
ENTRYPOINT ["/git-proxy"]