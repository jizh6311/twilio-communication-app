from kafka import KafkaProducer
from json import dumps

producer = KafkaProducer(
   value_serializer=lambda m: dumps(m).encode('utf-8'),
   bootstrap_servers=['localhost:9092'])

for i in range(5):
   producer.send("test", value={"hello": "producer-{}".format(i)})