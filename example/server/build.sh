#!/bin/sh
GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v github.com/huhuikevin/grpc-loadbalancer/example/server

if [ $? != 0 ];then
echo "build error"
exit 1
fi

docker build . -t rpcserver
rm -r server
if [ $? != 0 ];then
echo "build docker error"
exit 1
fi

docker tag rpcserver:latest docker.sharkgulf.cn/huhui/rpcserver:latest

docker push docker.sharkgulf.cn/huhui/rpcserver:latest
