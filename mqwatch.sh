#!/bin/bash

URL="amqp://${RABBITMQ_USAGE_SERVICE_HOST:=rabbitmq-usage}:${RABBITMQ_PORT:=5672}"
EXCHANGE="${RABBITMQ_EXCHANGE:=lenkung}"
PORT="${MQWATCH_SERVICE_PORT:=9090}"

echo "Starting mqwatch on $URL for exchange $EXCHANGE, listening for connections on $PORT"
./mqwatch -url "$URL" -exchange "$EXCHANGE" -port $PORT
