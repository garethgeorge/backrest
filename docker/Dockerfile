FROM golang:alpine as build

RUN mkdir /tmp-orig
COPY ./resticui-linux-amd64 /resticui-linux-amd64
RUN /resticui-linux-amd64 --install-only true 

FROM scratch
COPY --from=build /root/.local/share/resticui/ /.local/share/resticui/
COPY --from=build /tmp-orig /tmp
COPY ./resticui-linux-amd64 /resticui-linux-amd64

# WebUI port
EXPOSE 9898

ENTRYPOINT ["/resticui-linux-amd64"]
