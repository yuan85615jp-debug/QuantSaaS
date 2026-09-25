# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/saas ./cmd/saas \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/agent ./cmd/agent \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata bash curl python3
WORKDIR /app
COPY --from=build /out/saas /app/saas
COPY --from=build /out/agent /app/agent
COPY --from=build /out/seed /app/seed
COPY configs/config.yaml /app/configs/config.yaml
COPY configs/config.agent.example.yaml /app/configs/config.agent.example.yaml
COPY scripts/demo_paper.sh /app/scripts/demo_paper.sh
ENV QS_JWT_SECRET=change-me-in-production \
    QS_HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/saas", "-config", "/app/configs/config.yaml"]
