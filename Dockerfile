FROM golang:1.24 AS build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY tahvel tahvel
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /go/bin/teinetahvel

FROM gcr.io/distroless/static-debian12
LABEL org.opencontainers.image.source="https://github.com/jtagcat/teinetahvel"

COPY --from=build /go/bin/teinetahvel /
CMD ["/teinetahvel"]
EXPOSE 8080

COPY templates templates
