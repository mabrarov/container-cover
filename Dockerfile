FROM golang:1.23.6 AS build

COPY . /app

WORKDIR /app

ARG COVER_INSTRUMENT=0

RUN COVER_INSTRUMENT="${COVER_INSTRUMENT}" make build

FROM scratch

COPY --from=build /app/.build/app /app/

# Workaround to create /app/cover directory
WORKDIR /app/cover
WORKDIR /

ENTRYPOINT ["/app/app"]
