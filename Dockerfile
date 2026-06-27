FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -o /out/cachy ./cmd/cachy

FROM alpine:3.20

RUN adduser -D -H -s /sbin/nologin cachy
COPY --from=build /out/cachy /usr/local/bin/cachy

USER cachy
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -qO- http://127.0.0.1:8787/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/cachy"]
CMD ["proxy"]
