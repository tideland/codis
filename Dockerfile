##
## Build codis binary.
##
FROM golang:1.13 AS build
RUN mkdir /gocache
ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org
ENV CGO_ENABLED=0
WORKDIR /app
COPY . /app
RUN cd /app/cmd/codis && go build ./...

##
## Build runtime image.
##
FROM alpine AS production
WORKDIR /usr/bin
COPY --from=build /app/cmd/codis/codis .
ENV NAMESPACE=default RULENAME=default-rule
ENTRYPOINT ["/usr/bin/codis"]

##
## EOF
##