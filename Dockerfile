FROM golang:alpine as gobuild
RUN mkdir /tmp-orig
COPY backrest /backrest

FROM scratch 
COPY --from=gobuild /tmp-orig /tmp
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/backrest"]
COPY backrest /backrest