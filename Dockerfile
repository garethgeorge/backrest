FROM rclone/rclone:1.65 as rclone

FROM alpine:latest
COPY --from=rclone /usr/local/bin/rclone /usr/local/bin/rclone
RUN apk --no-cache add ca-certificates curl bash
RUN mkdir -p /tmp

ENTRYPOINT ["/backrest"]
COPY backrest /backrest