FROM golang:1.22

RUN groupadd -g 5000 searcher &&\
    useradd -u 5000 -g 5000 -d /app searcher &&\
    apt update &&\
    apt install -y ripgrep

WORKDIR /app

COPY go.mod go.sum ./
COPY config.yaml ./
RUN go mod download && go mod verify

COPY *.go ./

RUN go build -v -o ./searcher

USER searcher

CMD ["/app/searcher"]
