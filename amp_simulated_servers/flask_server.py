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
__version__ = '0.1.0'
import time

import jaeger_client
import json
import os
import random
import requests
import ruamel.yaml as yaml
from flask import Flask, Response, request
from flask_opentracing import FlaskTracer
from prometheus_flask_exporter.multiprocess import UWsgiPrometheusMetrics

app = Flask(__name__)
metrics = UWsgiPrometheusMetrics(app, defaults_prefix='amp_sim')
metrics.info('amp_simulated_server_info', 'Application info', version=__version__)

# configuration file: docker or local?
if os.path.exists("./configs/traffic_simulation.yml"):
    app.logger.info("Running locally from config file in ./configs...")
    config_file_path = "./configs/traffic_simulation.yml"
else:
    # dockers deploy yml to root application directory
    config_file_path = "./traffic_simulation.yml"
with open(config_file_path, "r") as of:
    ep = yaml.safe_load(of)
    app.logger.info("Loaded servers config file with keys={}".format(ep.keys()))

# tracing
def initialize_tracer():
    config = jaeger_client.Config(
        config={
            'sampler': {'type': 'const', 'param': 1},
            'local_agent': {
                'reporting_host': 'jaeger-agent',
                 'reporting_port': '6831'
            },
            'logging': True},
        service_name='amp-sim-server',
        validate=True)
    return config.initialize_tracer()  # also sets opentracing.tracer
flask_tracer = FlaskTracer(initialize_tracer, True, app)

##########################################
class Delay():
    """
    Base delay class for generating random response times
    """

    def __init__(self, endpoint="default", outlier_prob=0.0, **kwargs):
        self.endpoint = endpoint
        self.outlier_prob = outlier_prob
        if endpoint.endswith("_outlier"):
            key = endpoint[:-8]
            app.logger.debug('outlier endpoint, key={}'.format(key))
        else:
            key = endpoint
        if key in ep["endpoints"]:
            self.mu = ep["endpoints"][key]["mu"]
            self.sig = ep["endpoints"][key]["sig"]
        elif "default" in ep["endpoints"]:
            self.mu = ep["endpoints"]["default"]["mu"]
            self.sig = ep["endpoints"]["default"]["sig"]
        self.data = kwargs

    def delayed_response(self):
        """
        Use class parameters to (1) delay for a random time and (2) package up a json
        payload describing the delay  and return a flask response.
        """
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
    res, _ = Delay("/").delayed_response()
    return res


@app.route('/no_delay')
def no_delay():
    data = json.dumps({
        "endpoint": "no_delay",
        "params": {
            "mu": 0,
            "sig": 0},
        "delay": 0
    })
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)


@app.route('/item_type/<item_type>')
@metrics.do_not_track()
# only the counter is collected, not the default metrics
@metrics.counter('amp_sim_invocation_by_type', 'Number of invocations by type',
                 labels={'item_type': lambda: request.view_args['item_type']})
def by_type(item_type):
    data = json.dumps({
        "endpoint": "item_type",
        "item_type": item_type,
        "params": {
            "mu": 0,
            "sig": 0
        },
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
    res, dt = Delay("long_delay").delayed_response()
    return res


@app.route('/long_delay_outlier')
@metrics.gauge('amp_sim_long_delay_outlier_in_progress', 'Long running requests in progress')
def long_delay_outlier():
    res, dt = Delay("long_delay_outlier", outlier_prob=0.1).delayed_response()
    return res


@app.route('/short_delay')
@metrics.gauge('amp_sim_short_delay_in_progress', 'Short running requests in progress')
def short_delay():
    res, dt = Delay("short_delay").delayed_response()
    return res


@app.route('/short_delay_outlier')
@metrics.gauge('amp_sim_short_delay_outlier_in_progress', 'Short running requests in progress')
def short_delay_outlier():
    res, dt = Delay("short_delay_outlier", outlier_prob=0.1).delayed_response()
    return res


@app.route('/status/<int:status>')
@metrics.do_not_track()
@metrics.summary('amp_sim_requests_by_status', 'Request latencies by status',
                 labels={'status': lambda r: r.status_code})
@metrics.histogram('requests_by_status_and_path', 'Request latencies by status and path',
                   labels={'status': lambda r: r.status_code, 'path': lambda: request.path})
def echo_status(status):
    res, _ = Delay("echo_status", status=status).delayed_response()
    return res


@app.route('/downstream/<service_type>')
@metrics.do_not_track()
@metrics.counter('amp_sim_downstream_by_type', 'Number of invocations by downstream service type',
                 labels={'service_type': lambda: request.view_args['service_type']})
def downstream_service(service_type):
    if "downstream" in ep["endpoints"]:
        url = "http://{}:{}/{}".format(
            ep["endpoints"]["downstream"]["host"],
            ep["endpoints"]["downstream"]["port"],
            service_type)
        parent_span = flask_tracer.get_span()
        with flask_tracer.tracer.start_span(
                "downstream-{}".format(service_type),
                child_of=parent_span) as span:
            span.set_tag("http.url", url)
            result = requests.get(url)
            span.set_tag("http.status_code", result.status_code)
            result = result.json()
    else:
        result = {"msg": "configuration problem, no downstream request processed"}
    res, _ = Delay("downstream", service_type=service_type, downstream=result).delayed_response()
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
