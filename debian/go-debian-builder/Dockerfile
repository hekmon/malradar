FROM golang:latest
ADD start.sh /usr/local/bin/start
RUN chmod +x /usr/local/bin/start
RUN apt-get update && apt-get install -y --no-install-recommends debhelper build-essential && apt-get clean
CMD [ "/usr/local/bin/start" ]
