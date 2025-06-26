
FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .


RUN go build -o /app/main .

# Expõe a porta que nossa aplicação usa
EXPOSE 8080

CMD ["/app/main"]