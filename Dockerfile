FROM golang:latest AS build
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static:latest
COPY --from=build /go/bin/app /

CMD ["/app"]