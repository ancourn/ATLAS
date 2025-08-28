FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/inboxsvc ./cmd/inboxsvc
FROM alpine:latest
COPY --from=build /bin/inboxsvc /bin/inboxsvc
EXPOSE 9200
ENTRYPOINT ["/bin/inboxsvc"]