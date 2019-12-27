##
## Build codis binary.
##
FROM golang:1.13 AS build
COPY go.mod /src/
COPY go.sum /src/
RUN cd /src && go mod download
COPY cmd /src/cmd
COPY pkg /src/pkg
RUN cd /src && go build -mod=readonly -o /usr/local/bin/codis ./cmd/codis
##
## Build runtime image.
##
FROM gcr.io/distroless/static:nonroot
USER nobody
COPY --from=build /usr/local/bin/codis /usr/local/bin/codis
ENV NAMESPACE=default RULENAME=default-rule
ENTRYPOINT ["/usr/local/bin/codis"]
