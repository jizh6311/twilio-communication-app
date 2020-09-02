from kafka import KafkaConsumer
from json import loads
import base64

topic_name = 'jaeger-spans'
group_id = None

class KafkaTrace:
    def __init__(self, value_dict):
        self._value_dict = value_dict

    @property
    def duration(self):
        return float(self._value_dict["duration"].strip("s"))

    @property
    def trace_id(self):
        message_bytes = base64.b64decode(self._value_dict["traceId"]).hex()
        return message_bytes

    @property
    def span_id(self):
        message_bytes = base64.b64decode(self._value_dict["spanId"]).hex()
        return message_bytes

    @property
    def tag_list(self):
        return self._value_dict["tags"]

    @property
    def span_dict(self):
        return {
            "spanID": self.span_id,
            "traceID": self.trace_id,
            "duration": self.duration,
            "tags": self.tag_list
        }

if __name__ == "__main__":
    consumer = KafkaConsumer(topic_name,
                             group_id=group_id,
                             bootstrap_servers=['localhost:9093'],
                             value_deserializer=lambda m: loads(m))

    print("Consuming topic = '{}' with group_id = '{}'".format(topic_name, group_id))

    for message in consumer:
        # message value and key are raw bytes -- decode if necessary!
        # e.g., for unicode: `message.value.decode('utf-8')`
        print ("%s:%d:%d: key=%s value=%s" % (message.topic, message.partition,
                                              message.offset, message.key,
                                              message.value))
        kt = KafkaTrace(message.value)
        print(kt.span_dict)

    print("Quit")
