FROM golang:1.24.5 AS build

WORKDIR /go/src/app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
ENV CGO_ENABLED=0
RUN go mod download

# Copy the rest of the source code
COPY . .
RUN go build -o /go/bin/app cmd/app/*.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /go/bin/app /
CMD ["/app"]
