FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/docsvc ./cmd/docsvc
FROM alpine:latest
COPY --from=build /bin/docsvc /bin/docsvc
EXPOSE 9300
ENTRYPOINT ["/bin/docsvc"]