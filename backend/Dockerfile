# The API and the seed command for running MAPP locally with compose (ADR-0012). Not a production image.

FROM golang:1.27.3-alpine AS build
WORKDIR /src
# Modules first, so a change to the code alone reuses this layer.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/seed

# No shell or package manager; runs as the image's non-root user (65532).
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /out/seed /usr/local/bin/
ENV APP_HOST=0.0.0.0
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
