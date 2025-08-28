FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/transcribesvc ./cmd/transcribesvc
FROM alpine:latest
COPY --from=build /bin/transcribesvc /bin/transcribesvc
EXPOSE 9501
ENTRYPOINT ["/bin/transcribesvc"]