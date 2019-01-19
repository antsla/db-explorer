FROM golang:latest

ADD . /home

WORKDIR /home

RUN go get github.com/go-sql-driver/mysql
RUN go get github.com/gorilla/mux

ENTRYPOINT ["sh", "/entrypoint.sh"]

EXPOSE 8084