"""
Service with random delays at various endpoints.  Instrumented with Prometheus and Tracing.

From root of repo:
    docker build . -f amp_simulated_servers/Dockerfile-simulated-servers --tag amp-sim-server:1.0
    docker run --publish 5000:7599 amp-sim-server:1.0

Either disconnect or from another terminal:
    poetry run python bin/simulated_traffic_client.py

Metrics at:
    http://localhost:5000/metrics
"""
import time
import os
import json
import random
import ruamel.yaml as yaml
from flask import Flask, Response, request
from prometheus_flask_exporter.multiprocess import UWsgiPrometheusMetrics

app = Flask(__name__)
metrics = UWsgiPrometheusMetrics(app, defaults_prefix='amp_sim')

# static information as metric
metrics.info('amp_simulated_server_info', 'Application info', version='0.0.3')

# docker or local?
if os.path.exists("./configs/flask_server.yml"):
    app.logger.info("Running locally from config file in ./configs...")
    config_file_path = "./configs/flask_server.yml"
else:
    config_file_path = "./flask_server.yml"

with open(config_file_path, "r") as of:
    ep = yaml.safe_load(of)
    app.logger.info("Loaded servers config file with keys={}".format(ep.keys()))


class delay():
    """
    Base delay class for generating random response times
    """

    def __init__(self, endpoint="default", outlier_prob=0.0, **kwargs):
        self.endpoint = endpoint
        self.outlier_prob = outlier_prob
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
            # outlier is 6 sigma
            t *= 6.0
        time.sleep(t)
        app.logger.info("endpoint={};mu={};sig={};t={}".format(self.endpoint, self.mu, self.sig, t))
        self.data.update({
            "endpoint": self.endpoint,
            "params": {
                "mu": self.mu,
                "sig": self.sig},
            "delay": t
        })
        rdata = json.dumps(self.data)
        response_headers = [
            ('Content-type', 'application/json'),
            ('Content-Length', str(len(rdata)))
        ]
        return Response(response=rdata, status=200, headers=response_headers), t

    def _wait(self):
        """overwrite this for other distributions"""
        return random.lognormvariate(self.mu, self.sig)


#################################################
@app.route('/')
def main():
    res, _ = delay("/").delayed_response()
    return res


@app.route('/no_delay')
def no_delay():
    data = json.dumps({
        "endpoint": "no_delay",
        "params": {"mu": 0, "sig": 0},
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
        "params": {"mu": 0, "sig": 0},
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
    res, dt = delay("long_delay").delayed_response()
    return res


@app.route('/long_delay_outlier')
@metrics.gauge('amp_sim_long_delay_outlier_in_progress', 'Long running requests in progress')
def long_delay_outlier():
    res, dt = delay("long_delay_outlier", outlier_prob=0.1).delayed_response()
    return res


@app.route('/short_delay')
@metrics.gauge('amp_sim_short_delay_in_progress', 'Short running requests in progress')
def short_delay():
    res, dt = delay("short_delay").delayed_response()
    return res


@app.route('/short_delay_outlier')
@metrics.gauge('amp_sim_short_delay_outlier_in_progress', 'Short running requests in progress')
def short_delay_outlier():
    res, dt = delay("short_delay_outlier", outlier_prob=0.1).delayed_response()
    return res


@app.route('/status/<int:status>')
@metrics.do_not_track()
@metrics.summary('amp_sim_requests_by_status', 'Request latencies by status',
                 labels={'status': lambda r: r.status_code})
@metrics.histogram('requests_by_status_and_path', 'Request latencies by status and path',
                   labels={'status': lambda r: r.status_code, 'path': lambda: request.path})
def echo_status(status):
    res, _ = delay("echo_status", status=status).delayed_response()
    return res


#################################################

@app.route('/metrics')
def metrics():
    """
    Creates the metrics endpoint for prometheus scrapes
    """
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
