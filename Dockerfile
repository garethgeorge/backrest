FROM golang:alpine as gobuild
RUN mkdir /tmp-orig
COPY backrest /backrest
RUN /backrest --install-deps=true 

FROM scratch 
COPY --from=gobuild /root/.local/share/backrest/ /.local/share/backrest/
COPY --from=gobuild /tmp-orig /tmp
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/backrest"]
COPY backrest /backrest