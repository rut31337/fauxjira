FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /fauxjira .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /fauxjira /fauxjira
RUN mkdir -p /data
EXPOSE 6778
ENTRYPOINT ["/fauxjira"]
