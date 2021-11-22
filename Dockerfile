FROM golang:alpine AS build-env
COPY . /work
WORKDIR /work
RUN go build

FROM alpine
WORKDIR /sxbin
COPY --from=build-env /work/robot_provider /sxbin/robot_provider
ENTRYPOINT ["/sxbin/robot_provider", "--nodesrv", "nodeserv:9990"]
CMD [""]