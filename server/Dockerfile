FROM golang:latest as builder

LABEL maintainer="Carson Hoffman <hoffcar@umich.edu>"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ./main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

# This is a bit hacky; on initialize the DB container isn't instantly
# ready, so even though we depend on it we might panic if we start too soon.
CMD ["sleep", "10"]
CMD ["./main"]
