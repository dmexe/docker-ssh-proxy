FROM golang:1.8

RUN mkdir /app
ADD . /app
WORKDIR /app
CMD ["bin/docker-build", "build"]
