FROM golang:1.10

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

ENTRYPOINT ["dep"]
