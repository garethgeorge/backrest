FROM golang:alpine as gobuild
RUN mkdir /tmp-orig
COPY restora /restora
RUN /restora --install-deps=true 

FROM scratch 
COPY --from=gobuild /root/.local/share/restora/ /.local/share/restora/
COPY --from=gobuild /tmp-orig /tmp
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/restora"]
COPY restora /restora