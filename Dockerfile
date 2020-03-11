FROM golang:1.14-alpine AS build

WORKDIR /src/
COPY . .
RUN chmod 777 /src/items.json
RUN CGO_ENABLED=0 go build -o /bin/bidtracker

FROM scratch
COPY --from=build /bin/bidtracker /bin/bidtracker
COPY --from=build /src/items.json items.json

ENTRYPOINT ["/bin/bidtracker"]
