FROM alpine:3.14.2
LABEL version="1.0.2"
LABEL alpineversion="3.14.2"
LABEL golangversion="1.17.0"
LABEL homepage="https://github.com/hekmon/malradar"
ENV VIGIXPORTER_STATIONS=""
COPY malradar_alpine /usr/local/bin/malradar
VOLUME /var/lib/malradar
WORKDIR /var/lib/malradar
ENTRYPOINT /usr/local/bin/malradar -conf /etc/malradar/config.json