from kafka import KafkaConsumer
from json import loads

topic_name = 'jaeger-spans'
#topic_name = 'test'
group_id = 'my_group_id'
group_id = None

# To consume latest messages and auto-commit offsets

consumer = KafkaConsumer(topic_name,
                         group_id=group_id,
                         bootstrap_servers=['localhost:9093'],
                         value_deserializer=lambda m: loads(m.decode('utf-8')))

print("Consuming topic = '{}' with group_id = '{}'".format(topic_name, group_id))

for message in consumer:
    # message value and key are raw bytes -- decode if necessary!
    # e.g., for unicode: `message.value.decode('utf-8')`
    print ("%s:%d:%d: key=%s value=%s" % (message.topic, message.partition,
                                          message.offset, message.key,
                                          message.value))

print("Quit")
