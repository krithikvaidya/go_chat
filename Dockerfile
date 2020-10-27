FROM golang:1.15

WORKDIR /go_chat

COPY . ./

EXPOSE 4545

RUN go install .

CMD go_chat -s -p "password"