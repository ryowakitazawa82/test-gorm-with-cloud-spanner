FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN GGO_ENABLED=0 GOOS=linux go build -o main

FROM openjdk:slim-buster AS runner
RUN apt update && apt -y install curl supervisor
RUN curl -sL https://storage.googleapis.com/pgadapter-jar-releases/pgadapter.tar.gz \
  | tar xzf - -C /
COPY --from=builder /app/main /main
COPY supervisord.conf /etc/supervisor/supervisord.conf
CMD ["supervisord"]
