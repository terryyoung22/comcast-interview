FROM golang:1.25-alpine
WORKDIR /app

COPY app/ .
COPY samples ./samples
#^ optional, you can use the "-v" command to run it without copying the samples

RUN go mod download
RUN go build -o app .
RUN chmod +x ./app

ENTRYPOINT ["./app"]