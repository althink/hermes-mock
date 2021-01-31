FROM scratch

MAINTAINER Krzysztof Szczesniak "k.szczesniak@althink.com"

EXPOSE 8080

COPY hermes-stub /app/hermes-stub

CMD ["/app/hermes-stub"]