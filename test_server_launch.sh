#!/bin/bash
set -e  # どれかのコマンドが失敗したら終了
set -o pipefail

cleanup(){
    echo "cleaning up..."
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        echo "Server process killed."
    fi
}

trap cleanup EXIT

echo "Starting server"
go run . serve ../s3_baby_server_test --addr 127.0.0.1:7000 --logPath ../s3_baby_server_test/test_server.log&
SERVER_PID=$!

sleep 2

echo "サーバー立ち上げ完了"

bash test_aws_cli.sh

rm -r download.txt
rm -r test.txt

echo "サーバー停止"