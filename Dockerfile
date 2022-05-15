FROM golang:alpine AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal

RUN go build -o app ./cmd/shared-planner

FROM alpine
LABEL maintainer="Sergey Kozhin <kozhinsergeyv@gmail.com>"
LABEL org.opencontainers.image.source=https://github.com/SergeyKozhin/shared-planner

WORKDIR /app

COPY --from=builder /build/app .
EXPOSE 80
CMD ["./app"]
