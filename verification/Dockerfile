FROM golang:1.17 AS Build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM alpine

WORKDIR /app

COPY --from=Build /src/app .

CMD [ "./app" ]