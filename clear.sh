#/bin/sh

if test -e RtdServer; then
    rm -f RtdServer
fi
if test -e HisServer; then
    rm -f HisServer
fi
if test -e Simulator; then
    rm -f Simulator
fi
if test -e RabbitMQSendTest; then
    rm -f RabbitMQSendTest
fi
if test -e RabbitMQRecvTest; then
    rm -f RabbitMQRecvTest
fi