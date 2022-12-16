FROM golang:1.19 as build
WORKDIR /build
COPY . /build
RUN go install .

FROM gcr.io/distroless/base:debug
WORKDIR /
ENTRYPOINT [ "procfly" ]
COPY --from=build /go/bin/procfly /bin/procfly
