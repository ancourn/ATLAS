FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/ai-router ./cmd/ai-router
FROM alpine:latest
COPY --from=build /bin/ai-router /bin/ai-router
EXPOSE 9400
ENTRYPOINT ["/bin/ai-router"]