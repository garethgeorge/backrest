FROM golang:alpine as gobuild
RUN mkdir /tmp-orig
COPY backrest /backrest

FROM rclone/rclone:1.65 as rclone

FROM alpine:latest
COPY --from=gobuild /tmp-orig /tmp
COPY --from=rclone /usr/local/bin/rclone /usr/local/bin/rclone
RUN apk --no-cache add ca-certificates bash curl

ENTRYPOINT ["/backrest"]
COPY backrest /backrest