# One image builds every service binary into /app. compose.yaml picks which one
# each container runs via `command`.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/...

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/ /app/
