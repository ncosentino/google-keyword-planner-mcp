FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go/go.mod go/go.sum ./
RUN go mod download

COPY go/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /google-keyword-planner-mcp .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /google-keyword-planner-mcp /google-keyword-planner-mcp

EXPOSE 8080

ENTRYPOINT ["/google-keyword-planner-mcp", "--transport", "http"]
