FROM golang:latest AS base

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make build

EXPOSE 8242

CMD ["/build/bin/musannif --serve"]
