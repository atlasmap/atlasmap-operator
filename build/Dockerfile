FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/atlasmap-operator /usr/local/bin/atlasmap-operator
