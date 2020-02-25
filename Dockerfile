FROM golang:latest AS build-env
RUN apt update; apt install -y libvips-dev
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o main .
RUN apt remove -y libvips-dev; apt autoremove -y
RUN apt install libjpeg62-turbo libvips -y
CMD ["/app/main"]