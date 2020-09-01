from kafka import KafkaProducer
from json import dumps

topic_name = 'jaeger-spans'
topic_name = 'test'

producer = KafkaProducer(
   value_serializer=lambda m: dumps(m).encode('utf-8'),
   bootstrap_servers=['localhost:9093'])

# Asynchronous by default
future = producer.send(topic_name, {"list":[1,2,3]})

# Block for 'synchronous' sends
try:
   record_metadata = future.get(timeout=10)
except KafkaError:
   # Decide what to do if produce request failed...
   print("get failed")

# Successful result returns assigned partition and offset
print("topic={}".format(record_metadata.topic))
print("partition={}".format(record_metadata.partition))
print("offset={}".format(record_metadata.offset))

for i in range(5):
   producer.send(topic_name, value={"hello": "producer-{}".format(i)})

# block until all async messages are sent
producer.flush()
