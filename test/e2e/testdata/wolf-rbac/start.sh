#!/bin/sh
cd testdata/wolf-rbac/

# start database
docker-compose up -d database

# start wolf-server
docker-compose up -d server restful-demo agent-or agent-demo

sleep 6

WOLF_TOKEN=`curl http://127.0.0.1:12180/wolf/user/login  -H "Content-Type: application/json"  -d '{ "username": "root", "password": "wolf-123456"}' -s | grep token| tr -d ':",' | awk '{print $2}'`


curl http://127.0.0.1:12180/wolf/application \
-H "Content-Type: application/json" \
-H "x-rbac-token: $WOLF_TOKEN" \
-d '{
    "id": "test-app", 
    "name": "application for test"
}'


curl http://127.0.0.1:12180/wolf/resource \
-H "Content-Type: application/json" \
-H "x-rbac-token: $WOLF_TOKEN" \
-d '{
    "appID": "test-app",
    "matchType": "prefix",
    "name": "/",
    "action": "GET",
    "permID": "ALLOW_ALL"
}'

curl http://127.0.0.1:12180/wolf/user \
-H "Content-Type: application/json" \
-H "x-rbac-token: $WOLF_TOKEN" \
-d '{
    "username": "test",
    "nickname": "test",
    "password": "test-123456",
    "appIDs": ["test-app"]
}'

