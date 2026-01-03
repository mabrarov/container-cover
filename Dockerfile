FROM golang:1.24.9 AS build

COPY . /app

WORKDIR /app

ARG COVER_INSTRUMENT=0

RUN COVER_INSTRUMENT="${COVER_INSTRUMENT}" make build

FROM scratch

COPY --from=build /app/.build/app /app/

ENTRYPOINT ["/app/app"]
