#!/bin/bash

MAX_ATTEMPTS=20

WORKDIR=$(dirname $(realpath $0))

docker-compose up -d postgres rabbitmq
ATTEMPT=1
while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    echo "Attempt $ATTEMPT of $MAX_ATTEMPTS"
    if docker-compose exec postgres psql -U postgres -d gearr -c "SELECT 1" &> /dev/null; then
        echo "postgres running"
        break
    fi
    sleep 1
    ((ATTEMPT++))
done

ATTEMPT=1
while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    echo "Attempt $ATTEMPT of $MAX_ATTEMPTS"
    if docker-compose exec rabbitmq rabbitmqctl status &> /dev/null; then
        echo "rabbitmq running"
        break
    fi
    if [ $ATTEMPT = 10 ]; then
        docker-compose up -d rabbitmq
    fi
    sleep 1
    ((ATTEMPT++))
done

docker-compose up -d

ATTEMPT=1
while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    echo "Attempt $ATTEMPT of $MAX_ATTEMPTS"
    if curl -s http://localhost:8080/-/healthy &> /dev/null; then
        echo "Server running"
        break
    fi
    sleep 1
    ((ATTEMPT++))
done

if [ -n "$NOT_RUN_FRONT" ]; then
    exit 0
fi

cd $WORKDIR/../server/web/ui && npm start
