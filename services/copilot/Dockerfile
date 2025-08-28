FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/copilotsvc ./cmd/copilotsvc
FROM alpine:latest
COPY --from=build /bin/copilotsvc /bin/copilotsvc
EXPOSE 9502
ENTRYPOINT ["/bin/copilotsvc"]