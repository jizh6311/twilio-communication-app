import time
import random
import json
import ruamel.yaml as yaml
from contextlib import contextmanager

from flask import Flask, request, Response
from prometheus_flask_exporter.multiprocess import UWsgiPrometheusMetrics

app = Flask(__name__)
metrics = UWsgiPrometheusMetrics(app, defaults_prefix='amp_sim')

# static information as metric
metrics.info('amp_simulated_server_info', 'Application info', version='0.0.2')

#################################################
# utilities for random time delays
with open("./flask_server.yml", "r") as of:
    ep = yaml.safe_load(of)
    app.logger.info("read servers config file with keys={}".format(ep.keys()))

class delay():
    def __init__(self, endpoint="default", outlier_prob=0.0, **kwargs):
        self.endpoint = endpoint
        self.outlier_prob = outlier_prob    # outlier is 10*t
        if self.endpoint in ep["endpoints"]:
            self.mu = ep["endpoints"][self.endpoint]["mu"]
            self.sig = ep["endpoints"][self.endpoint]["sig"]
        elif "default" in ep["endpoints"]:
            self.mu = ep["endpoints"]["default"]["mu"]
            self.sig = ep["endpoints"]["default"]["sig"]
        self.data = kwargs

    def delayed_response(self):
        if self.sig == 0.0:
            t = self.mu
        else:
            t = self._wait()
        if random.random() < self.outlier_prob:
            t *= 10.0
        time.sleep(t)
        app.logger.info("endpoint={};mu={};sig={};t={}".format(self.endpoint, self.mu, self.sig, t))
        self.data.update({
            "endpoint": self.endpoint,
            "params": {
                    "mu": self.mu,
                    "sig": self.sig },
            "delay": t
        })
        rdata = json.dumps(self.data)
        response_headers = [
            ('Content-type', 'application/json'),
            ('Content-Length', str(len(rdata)))
        ]
        return Response(response=rdata, status=200, headers=response_headers)

    def _wait(self):
        """overwrite this for other distributions"""
        return random.lognormvariate(self.mu, self.sig)

#################################################

@app.route('/')
def main():
    return delay("/").delayed_response()


@app.route('/no_delay')
def no_delay():
    data = json.dumps({
        "endpoint": "no_delay",
        "delay": 0
    })
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)


@app.route('/item_type/<item_type>')
@metrics.do_not_track()
@metrics.counter('amp_sim_invocation_by_type', 'Number of invocations by type',
         labels={'item_type': lambda: request.view_args['item_type']})
def by_type(item_type):
    # only the counter is collected, not the default metrics
    data = json.dumps({
        "endpoint": "item_type",
        "item_type": item_type,
        "delay": 0
    })
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)


@app.route('/long_delay')
@metrics.gauge('amp_sim_long_delay_in_progress', 'Long running requests in progress')
def long_delay():
    return delay("long_delay").delayed_response()

@app.route('/long_delay_outlier')
@metrics.gauge('amp_sim_long_delay_outlier_in_progress', 'Long running requests in progress')
def long_delay_outlier():
    return delay("long_delay", outlier_prob=0.1).delayed_response()

@app.route('/short_delay')
@metrics.gauge('amp_sim_short_delay_in_progress', 'Short running requests in progress')
def short_delay():
    return delay("short_delay").delayed_response()

@app.route('/short_delay_outlier')
@metrics.gauge('amp_sim_short_delay_outlier_in_progress', 'Short running requests in progress')
def short_delay_outlier():
    return delay("short_delay", outlier_prob=0.1).delayed_response()


@app.route('/status/<int:status>')
@metrics.do_not_track()
@metrics.summary('amp_sim_requests_by_status', 'Request latencies by status',
                 labels={'status': lambda r: r.status_code})
@metrics.histogram('requests_by_status_and_path', 'Request latencies by status and path',
                   labels={'status': lambda r: r.status_code, 'path': lambda: request.path})
def echo_status(status):
    return delay("echo_status", status=status).delayed_response()

#################################################

@app.route('/metrics')
def metrics():
    from prometheus_client import multiprocess, CollectorRegistry, generate_latest, CONTENT_TYPE_LATEST
    registry = CollectorRegistry()
    multiprocess.MultiProcessCollector(registry)
    data = generate_latest(registry)
    response_headers = [
        ('Content-type', CONTENT_TYPE_LATEST),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
