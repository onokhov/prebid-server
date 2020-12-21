FROM golang:alpine AS build
RUN mkdir -p /app/prebid-server/
WORKDIR /app/prebid-server/
#ENV GOPROXY="https://proxy.golang.org"
#RUN apt-get update && \
#    apt-get install -y git && \
#    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV CGO_ENABLED 0
COPY ./ ./
RUN go mod vendor
RUN go mod tidy
ARG TEST="true"
RUN ls -la
RUN if [ "$TEST" != "false" ]; then /bin/sh validate.sh ; fi
RUN go build -mod=vendor .

FROM ubuntu:18.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-server/prebid-server .
COPY static static/
COPY stored_requests/data stored_requests/data
RUN apt-get update && \
    apt-get install -y ca-certificates mtr && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
EXPOSE 8000
EXPOSE 6060
ENTRYPOINT ["/usr/local/bin/prebid-server"]
CMD ["-v", "1", "-logtostderr"]
