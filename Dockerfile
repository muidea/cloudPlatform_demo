FROM alpine

ADD HisServer /usr/local/cloundPlatform/
ADD RtdServer /usr/local/cloundPlatform/
ADD Simulator /usr/local/cloundPlatform/


ENTRYPOINT ["/usr/local/cloundPlatform/HisServer","-Rabbitmq='amqp://guest:guest@localhost:5672/'"]
