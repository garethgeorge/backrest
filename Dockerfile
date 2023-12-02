FROM golang:alpine as gobuild
RUN mkdir /tmp-orig
COPY resticweb /resticweb
RUN /resticweb --install-deps=true 

FROM scratch 
COPY --from=gobuild /root/.local/share/resticui/ /.local/share/resticui/
COPY --from=gobuild /tmp-orig /tmp

ENTRYPOINT ["/resticweb"]
COPY resticweb /resticweb