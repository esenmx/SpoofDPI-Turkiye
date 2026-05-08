FROM golang:1.26.3-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-w -s -extldflags "-static"' -tags timetzdata -o /out/spoofdpi .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/spoofdpi /spoofdpi
USER 65532:65532
ENTRYPOINT ["/spoofdpi"]
