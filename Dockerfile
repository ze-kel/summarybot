FROM golang:1.22 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd ./cmd

RUN GOOS=linux go build -o /bot ./cmd

FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /app

COPY --from=build-stage /bot /bot

ENTRYPOINT ["/bot"]
