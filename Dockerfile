FROM centurylink/ca-certs

COPY endpoints /
COPY certs /

EXPOSE 13800

ENTRYPOINT ["/endpoints"]
