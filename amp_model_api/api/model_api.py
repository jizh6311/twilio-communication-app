import json

from flask import Flask, request, Response, Blueprint
from prometheus_flask_exporter.multiprocess import UWsgiPrometheusMetrics

import ruamel.yaml as yaml
import redis

from amp_prometheus.prometheus_query import *
from amp_prometheus.prometheus_collection_factory import ClusterMetricsCollectionFactory, TypeStrategy

av1 = Blueprint('amp_model_api', __name__, template_folder='templates')
app = Flask(__name__)

prom_metrics = UWsgiPrometheusMetrics(app, defaults_prefix='amp_model')
prom_metrics.info('amp_model_api_info', 'Model Application Info', version='0.0.1')

# model configuration from file. This must succeed.
with open("./model_api.yml", "r") as of:
    model_config = yaml.safe_load(of)
    app.logger.info("read servers config file with keys={}".format(model_config.keys()))

# extract the local url/cred configs (used by prometheus_query
config = model_config["prometheus_query"]["simulation"]

redis_inst = redis.StrictRedis(host='redis-server', port=6379, db=0,
                               decode_responses=True,password="devpassword")
redis_inst.client_setname("amp_model_api")

##########################################

@av1.route('/configuration')
def info():
    # listing of metrics
    config_json = {
        "model_config": model_config,
        "redis": {
            "connection_id": redis_inst.client_id(),
            "client_list": redis_inst.client_list(),
            "server_info": redis_inst.info()
                  }
    }
    data = json.dumps(config_json)
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)

@av1.route('/prototypes')
def prototypes():
    cmf = ClusterMetricsCollectionFactory(config, TypeStrategy())
    data = "<pre>" + cmf.show_metric_prototype() + "</pre>"
    response_headers = [
        ('Content-type', 'text/html'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)

@av1.route('/query')
def query():
    labels_str = None
    labels_dict = {}
    query = request.args.get('query')
    app.logger.debug("query={}".format(query))
    if "{" in query:
        metric_name = query.split("{")[0]
        labels_str = query.split("{")[1].split("}")[0]
    else:
        metric_name = query.strip(' "')

    app.logger.debug("labels_str={}".format(labels_str))

    if labels_str is not None and labels_str != "":
        for x in labels_str.split(","):
            [a,b] = x.split("=")
            labels_dict[a.strip(' "')] = b.strip(' "')

    cmf = ClusterMetricsCollectionFactory(config, TypeStrategy())
    mc = cmf.create_metric_collection(metric_name).query(arguments=labels_dict,period="5m")
    a, d = mc.analyze_collection(plots=False, output=False)

    out = { "name": metric_name,
            "query_labels": labels_dict,
            "aggregated": a,
            "diagnostic": d}
    data = json.dumps(out, cls=MetricEncoder)
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)

@av1.route('/outlier_scores')
def outlier_scores():
    n = int(request.args.get('n', 10))

    res = redis_inst.hgetall("outlier_scores")
    data_list = json.loads(res['data'])[:n]
    timestamp = res['timestamp']
    data = json.dumps({"timestamp": timestamp, "data": data_list})
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)

@av1.route('/increment_counter')
def increment_counter():
    from prometheus_client import multiprocess, CollectorRegistry, Counter
    registry = CollectorRegistry()
    multiprocess.MultiProcessCollector(registry)

    sctr = Counter("synthetic_test_counter",
            "Total inc of the synthetic counter",
            ('a','b'),
            registry=registry)
    sctr.labels("monday", "tuesday").inc()
    app.logger.info("test counter incremented")
    response_headers = [
        ('Content-type', 'text/html')
    ]
    return Response(response="test_counter incremented", status=200, headers=response_headers)

# register after definitions
app.register_blueprint(av1, url_prefix='/api/v1')

##########################################

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
    app.run(host="0.0.0.0", port=9545)
