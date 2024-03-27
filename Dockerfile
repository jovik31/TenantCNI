FROM golang:alpine as builder

WORKDIR /tenant
COPY go.mod go.sum ./

RUN go mod download

COPY . .



#Build images
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tenantcnid cmd/tenantcnid/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tenantcni cmd/tenantcni/main.go



FROM alpine:latest
RUN apk update && apk add --no-cache iptables

WORKDIR /
COPY --from=builder /tenant/bin/* /

