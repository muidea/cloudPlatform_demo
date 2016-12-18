FROM alpine

ADD HisServer /usr/local/cloudPlatform/
ADD RtdServer /usr/local/cloudPlatform/
ADD Simulator /usr/local/cloudPlatform/


ENTRYPOINT ["/usr/local/cloudPlatform/HisServer","-Rabbitmq='amqp://guest:guest@localhost:5672/'"]
