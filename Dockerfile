FROM golang:1.27-alpine AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 go build -o name-me -ldflags '-w -extldflags "-static"' ./cmd/...

FROM alpine:3.24.2

RUN apk add --no-cache ca-certificates

COPY --from=build /workspace/name-me /usr/local/bin/name-me

ENTRYPOINT ["name-me"]
