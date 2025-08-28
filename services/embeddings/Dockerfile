FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/embsvc ./cmd/embsvc
FROM alpine:latest
COPY --from=build /bin/embsvc /bin/embsvc
EXPOSE 9500
ENTRYPOINT ["/bin/embsvc"]