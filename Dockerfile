FROM golang:alpine AS builder
WORKDIR /src/app
COPY go.mod main.go ./
COPY index.html ./
COPY ./static ./static

RUN go mod tidy

RUN go build -o fax

FROM alpine
WORKDIR /root/
COPY --from=builder /src/app ./app
CMD ["./app/fax"]
