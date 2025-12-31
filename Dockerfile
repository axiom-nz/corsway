# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /corsway cmd/corsway/main.go

####

FROM gcr.io/distroless/base-debian13:nonroot AS release

EXPOSE 8080
USER nonroot

WORKDIR /
COPY --from=build /corsway /corsway

ENTRYPOINT ["/corsway"]
CMD []