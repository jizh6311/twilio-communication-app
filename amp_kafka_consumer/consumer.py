from kafka import KafkaConsumer
from json import loads

topic_name = 'test'

consumer = KafkaConsumer(topic_name,
                     bootstrap_servers=['localhost:9093'],
                     auto_offset_reset='earliest',
                     enable_auto_commit=True,
                     group_id=None,
                     value_deserializer=lambda x: loads(x.decode('utf-8')))

print("Consuming messages from the given topic")
for message in consumer:
    print("Message", message)
    if message is not None:
        print(message.offset, message.value)

print("Quit")




#
# consumer = KafkaConsumer(
#    'test',
#     auto_offset_reset='earliest',
#     enable_auto_commit=True,
#     group_id=None,
#     #group_id='my-group-1',
#     value_deserializer=lambda m: loads(m.decode('utf-8')),
#     bootstrap_servers=['localhost:9092'])
#
# for m in consumer:
#     print(m.value)