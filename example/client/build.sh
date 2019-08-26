#!/bin/sh
GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -v github.com/huhuikevin/grpc-loadbalancer/example/client

if [ $? != 0 ];then
echo "build error"
exit 1
fi

docker build . -t rpcclient
rm -r client

if [ $? != 0 ];then
echo "build docker error"
exit 1
fi

docker tag rpcclient:latest docker.sharkgulf.cn/huhui/rpcclient:latest

docker push docker.sharkgulf.cn/huhui/rpcclient:latest
