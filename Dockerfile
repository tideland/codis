##
## Build codis binary.
##
FROM golang:1.13 AS build
COPY go.mod /src/
COPY go.sum /src/
RUN mkdir /gocache
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org
RUN cd /src && go mod download
COPY cmd /src/cmd
COPY pkg /src/pkg
RUN cd /src && go build -mod=readonly ./cmd/codis

##
## Build runtime image.
##
FROM golang:1.13
WORKDIR /usr/bin
COPY --from=build /src/cmd/codis .
ENV NAMESPACE=default RULENAME=default-rule
ENTRYPOINT ["/usr/bin/codis"]

##
## EOF
##