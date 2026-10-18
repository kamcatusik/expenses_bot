FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /out/bot ./cmd/bot

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/bot /bot

CMD ["/bot"]
