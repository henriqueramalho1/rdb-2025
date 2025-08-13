FROM golang:1.24.5-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod/ \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o /bin/app ./cmd

FROM alpine:latest AS final

COPY --from=build /bin/app .

EXPOSE 8888

CMD ["./app"]