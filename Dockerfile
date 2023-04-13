FROM golang:1.20.3-bullseye

RUN mkdir /app

WORKDIR /app

ADD . /app

RUN go build -o Server ./myDemo/ZinxV1.0/Server/Server.go

EXPOSE 8080

CMD /app/Server