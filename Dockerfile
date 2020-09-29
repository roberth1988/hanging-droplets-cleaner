FROM alpine:3.6

ENV LISTEN 0.0.0.0:9380
ENV MACHINES_DIRECTORY /machines

EXPOSE 9380

RUN apk add -U ca-certificates; \
    mkdir -p $MACHINES_DIRECTORY

COPY ./hanging-droplets-cleaner /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/hanging-droplets-cleaner", "service"]
