FROM golang:1.23.6 AS build

COPY . /app

WORKDIR /app

RUN make build

FROM scratch

COPY --from=build /app/.build/app /app/

ENTRYPOINT ["/app/app"]
