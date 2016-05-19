FROM busybox:1.24.2

WORKDIR /app
ENTRYPOINT /app/reaper.exe

RUN addgroup -g 10001 app && \
    adduser -G app -u 10001 -D -h /app -s /sbin/nologin app

COPY version.json /app/version.json
COPY reaper.exe /app/reaper.exe

USER app