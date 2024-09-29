FROM docker.io/golang:1.23 AS builder
WORKDIR /src
COPY go.* .
RUN go mod download

COPY . /src/
RUN CGO_ENABLED=0 go build -o fax

# final image
FROM scratch
COPY --from=builder /src/fax /fax
CMD ["/fax"]
EXPOSE 3000
