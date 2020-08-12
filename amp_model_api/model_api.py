"""

    docker build . -f amp_model_api/Dockerfile-model-api --tag amp-model-api:1.0


To run locally:
    docker run --publish 6379:6379  redis:5.0-alpine --requirepass devpassword
    docker run --publish 9090:9090 prom/prometheus
    poetry run python amp_model_api/model_api.py

    http://localhost:9545/api/v1/query?query=prometheus_http_response_size_bytes_count
"""
import json
import redis
import ruamel.yaml as yaml
from amp_configuration.config import EndpointConfiguration
from amp_prometheus.prometheus_collection_factory import ClusterMetricsCollectionFactory, TypeStrategy
from amp_prometheus.prometheus_query import *
from flask import Blueprint, Flask, Response, request
from flask_opentracing import FlaskTracer
from jaeger_client import Config
from prometheus_flask_exporter.multiprocess import UWsgiPrometheusMetrics

av1 = Blueprint('amp_model_api', __name__, template_folder='templates')
app = Flask(__name__)

prom_metrics = UWsgiPrometheusMetrics(app, defaults_prefix='amp_model')
prom_metrics.info('amp_model_api_info', 'Model Application Info', version='0.0.1')

# docker or local?
if os.path.exists("./configs/traffic_simulation.yml"):
    logging.info("Running locally from config file in ./configs...")
    config_file_path = "./configs/traffic_simulation.yml"
    redis_host = "localhost"
    config_key = "local"
else:
    config_file_path = "./traffic_simulation.yml"
    redis_host = "redis-server"
    config_key = "simulation"
config = EndpointConfiguration(config_file_path, config_key)
with open(config_file_path, "r") as of:
    model_config = yaml.safe_load(of)
    app.logger.info("read servers config file with keys={}".format(model_config.keys()))


def initialize_tracer():
    config = Config(
        config={
            'sampler': {'type': 'const', 'param': 1},
            'local_agent': {
                'reporting_host': 'jaegertracing',
                'reporting_port': '6831',
            },
            'logging': True},
        service_name='amp-model-api',
        validate=True)
    return config.initialize_tracer()  # also sets opentracing.tracer
flask_tracer = FlaskTracer(initialize_tracer, True, app)

redis_inst = redis.StrictRedis(host=redis_host, port=6379, db=0,
                               decode_responses=True, password="devpassword")
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
    parent_span = flask_tracer.get_span()
    with flask_tracer.tracer.start_span(
            "prometheus_query",
            child_of=parent_span) as span:
        span.set_tag("query.type", "prototype")
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
            [a, b] = x.split("=")
            labels_dict[a.strip(' "')] = b.strip(' "')

    parent_span = flask_tracer.get_span()
    with flask_tracer.tracer.start_span(
            "prometheus_query",
            child_of=parent_span) as span:
        span.set_tag("query.type", "metric")
        cmf = ClusterMetricsCollectionFactory(config, TypeStrategy())
        mc = cmf.create_metric_collection(metric_name).query(arguments=labels_dict, period="5m")
    a, d = mc.analyze_collection(plots=False, output=False)

    out = {"name": metric_name,
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
    parent_span = flask_tracer.get_span()
    with flask_tracer.tracer.start_span(
            "outlier_scores_redis",
            child_of=parent_span) as span:
        span.set_tag("redis.host", redis_host)
        res = redis_inst.hgetall("outlier_scores")
        if 'data' in res:
            data_list = json.loads(res['data'])[:n]
            span.set_tag("redis.host.response", "ok")
        else:
            data_list = []
            span.set_tag("redis.host.response", "empty")
            span.set_tag("error", True)
    timestamp = res['timestamp']
    data = json.dumps({"timestamp": timestamp, "data": data_list})
    response_headers = [
        ('Content-type', 'application/json'),
        ('Content-Length', str(len(data)))
    ]
    return Response(response=data, status=200, headers=response_headers)


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
