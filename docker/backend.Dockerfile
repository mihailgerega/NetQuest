FROM golang:1.25-alpine AS build

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20

RUN addgroup -S netquest && adduser -S netquest -G netquest
WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY backend/migrations /app/migrations

USER netquest
EXPOSE 8080
CMD ["/bin/sh", "-c", "/app/migrate up && /app/api"]
