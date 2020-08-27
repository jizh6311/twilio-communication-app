from kafka import KafkaProducer
from json import dumps

producer = KafkaProducer(
   value_serializer=lambda m: dumps(m).encode('utf-8'),
   bootstrap_servers=['localhost:9092', '172.17.0.1:32783','172.17.0.1:32782','172.17.0.1:32781'])

producer.send("test", value={"hello": "producer"})