FROM centurylink/ca-certs

COPY endpoint_svc /
COPY certs /

EXPOSE 13800

ENTRYPOINT ["/endpoint_svc"]
