FROM alpine


COPY tenant /usr/local/bin/



ENTRYPOINT [ "tenant" ]