FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
# go.sum генерируется на сборке, чтобы не хранить его в репозитории
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.20
COPY --from=build /out/api /api
EXPOSE 8080
CMD ["/api"]
