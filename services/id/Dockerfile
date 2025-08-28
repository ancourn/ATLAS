FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/idsvc ./cmd/idsvc
FROM alpine:latest
COPY --from=build /bin/idsvc /bin/idsvc
EXPOSE 9100
ENTRYPOINT ["/bin/idsvc"]