FROM golang:1.26 AS builder
WORKDIR /app

COPY komputer-operator/ /komputer-operator/

COPY komputer-api/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o komputer-api .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/komputer-api .
USER 65532:65532
ENTRYPOINT ["/komputer-api"]
