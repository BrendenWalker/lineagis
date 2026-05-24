FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /verity-api ./cmd/verity-api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
COPY --from=build /verity-api /usr/local/bin/verity-api

EXPOSE 8080
USER nobody
ENTRYPOINT ["verity-api"]
