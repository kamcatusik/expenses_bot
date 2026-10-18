FROM golang:1.23-alpine AS build
WORKDIR /src

# Кэширование зависимостей: скачиваем модули отдельным слоем
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники и собираем бинарник с оптимизацией размера
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /api

EXPOSE 8080
USER 10001:10001
CMD ["/api"]
