import requests
import random
import threading
import time
import sys
import csv
import argparse

threadLock = threading.Lock()

class Client(threading.Thread):

    url = "http://localhost:5000/{}"
    endpoints = ["", "no_delay", "item_type/{}",
                 "long_delay", "short_delay",
                 "long_delay", "short_delay",
                 "long_delay", "short_delay",
                 "long_delay_outlier", "short_delay_outlier",
                 "status/{}"]
    choices = ["cat", "dog", "hat", "car", "peanut", "bird", "200", "300", "404", "500"]

    def __init__(self, threadID, name, delay, N=100):
        threading.Thread.__init__(self)
        self.threadID = threadID
        self.name = name
        self.delay = delay
        self.N = N

    def run(self):
        print ("Starting " + self.name)
        self.call_loop()
        print ("Exiting " + self.name)

    def call_loop(self):
        csvwrt = csv.writer(sys.stdout)
        for i in range(self.N):
            _tmp = random.choice(self.endpoints)
            if _tmp.endswith("{}"):
                u = self.url.format(_tmp.format(random.choice(self.choices)))
            else:
                u = self.url.format(_tmp)
            result = requests.get(u)
            output = [
                self.name, i, u, time.time(), result.status_code, result.text
            ]
            output = [str(o).strip().replace("\n"," ") for o in output]
            threadLock.acquire()
            csvwrt.writerow(output)
            threadLock.release()
            time.sleep(self.delay)

if __name__ == "__main__":

    parser = argparse.ArgumentParser()
    parser.add_argument("-v", "--verbose", help="increase output verbosity", action="store_true")
    parser.add_argument("-t", "--thread-pairs", help="number of client thread-pairs to launch", default=1)
    parser.add_argument("-n", "--slow-client-calls", help="number of slow-client api calls", default=100)
    args = parser.parse_args()

    clients = []
    for i in range(int(args.thread_pairs)):
        # Create new threads
        clients.append(Client(i, "Slow_client", 0.6, int(args.slow_client_calls)))
        clients.append(Client(i+1000, "Fast_client", 0.3, 2*int(args.slow_client_calls)))

    for c in clients:
        # Start new Threads
        c.start()

    for c in clients:
        c.join()

    print("Exiting Main Thread")
