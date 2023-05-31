FROM alpine

COPY ./simple-demo /
COPY ./printenv/printenv /

# Make sure the binaries have execute permissions
RUN chmod +x /simple-demo
RUN chmod +x /printenv

# Execute the binary
ENTRYPOINT ["/simple-demo"]
