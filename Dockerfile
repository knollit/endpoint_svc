FROM centurylink/ca-certs

COPY dest /
COPY certs /

EXPOSE 13800

ENTRYPOINT ["/endpoint_svc"]
