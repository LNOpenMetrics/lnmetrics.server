FROM golang:1.17-alpine3.13
LABEL maintainer="Vincenzo Palazzo (@vincenzopalazzo) vincenzopalazzodev@gmail.com"

WORKDIR lnmetrics

COPY . .

RUN apk add --update make && \
    go mod vendor && \   
    make prod

CMD ["./lnmetricsd"]