FROM golang:1.20.3-bullseye

RUN mkdir /app

WORKDIR /app

COPY . .

ADD . /app

RUN go build -o main ./server.go

EXPOSE 8080

CMD /app/myDemo/ZinxV1.0/Server/Server