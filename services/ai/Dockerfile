FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/aisvc ./cmd/aisvc
FROM alpine:latest
COPY --from=build /bin/aisvc /bin/aisvc
EXPOSE 9400
ENTRYPOINT ["/bin/aisvc"]