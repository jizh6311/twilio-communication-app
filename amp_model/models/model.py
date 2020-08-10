# coding: utf-8
import redis
import json
import time
import datetime
import logging
from operator import itemgetter

import ruamel.yaml as yaml

from amp_prometheus.prometheus_query import *
from amp_prometheus.prometheus_collection_factory import ClusterMetricsCollectionFactory
from amp_outliers.outliers import Outliers1d

redis_inst = redis.StrictRedis(host='redis-server', port=6379, db=0,
                               decode_responses=True,password="devpassword")
redis_inst.client_setname("amp_model-outlier_model")


class OutlierNodeSumCountStrategy:
    nonzero_limit = 30  # 1/nonzero_limit < percent_outlier_limit
    n_uniq_limit = 6  # diversity of sample values
    a = 5.5

    def outlier_score(self, v, w):
        if v is None:
            return (0, 0, 0, 0)
        v = v[v != 0]  # remove zeros
        v_out = Outliers1d.mad(v)  # outliers by distribution
        n = len(v)  # non-zero samples
        n_out = len(v[v_out])  # outliers
        n_uniq = len(set(v))  # count unique values
        if n < self.nonzero_limit or n_uniq < self.n_uniq_limit or n_out == 0:
            s = 0.0
        else:
            p_one_outlier = 1 / n
            p_out = n_out / n
            if p_out <= p_one_outlier:
                s = 1.0
            else:
                s = np.exp(-self.a * abs(p_out - p_one_outlier))
        result = (n, n_out, n_uniq, s)
        logging.debug("n={} n_out={}, n_uniq={}, s={:.4f}".format(*result))
        return (n, n_out, n_uniq, s)

    def analyze_metric(self, r, diag, agg, output_enabled, plots_enabled):
        (diag[-1]["n"],
         diag[-1]["n_out"],
         diag[-1]["uniq"],
         diag[-1]["outlier_score"]) = self.outlier_score(r.get_diff_values_vector(), diag[-1]["w"])
        logging.debug("processing {}".format(r.__name__))

    @staticmethod
    def metric_filter(l, w=1, os=0):
        res = []
        for x in l:
            if x["outlier_score"] > os and x["w"] > w:
                # remove the object reference and replace with string
                x["metric"] = x.__repr__()
                res.append(x)
            else:
                logging.debug("   ...metric {} dropped".format(x))
        return res


class OutlierTypeStrategy:
    def create_metric_collection(self, metric_name, prototype, config, metric_type)):
        """Determines the right Analysis strategy for each metric type and returns
        the appropriate object"""
        res = None
        if metric_name.endswith("sum") or metric_name.endswith("count") or metric_name.endswith("total"):
            res = MetricCollection(metric_name, config, OutlierNodeSumCountStrategy(), metric_type))
        return res

if __name__ == "__main__":

    logging.info("###############################################")
    logging.info("Starting model loop... ({})".format(datetime.datetime.utcnow()))
    logging.info("###############################################")

    # model configuration from file. This must succeed.
    with open("./model_api.yml", "r") as of:
        model_config = yaml.safe_load(of)
        logging.info("read servers config file with keys={}".format(model_config.keys()))

    # extract the local url/cred configs (used by prometheus_query
    config = model_config["prometheus_query"]["simulation"]
    delay = int(model_config["anomaly_detection_model"]["metric_loop"]["metric_delay"])

    while True:
        vm = ClusterMetricsCollectionFactory(config, OutlierTypeStrategy())
        logging.debug("==>analyzing {} metrics".format(len(vm)))
        res = []
        for x in vm.get_cluster_metrics_collection():
            logging.debug("-->query for {}".format(x.metric_name))
            x.query({}, "1h")
            a, d = x.analyze_collection(plots=False, output=False, w=0.1
            res.extend(x.metric_analysis_strategy_obj.metric_filter(d, w=0.1, os=-1))
        #
        res = sorted(res, key=itemgetter('outlier_score'), reverse=True)
        # put results into redis
        redis_inst.hset("outlier_scores", "timestamp", time.time())
        redis_inst.hset("outlier_scores", "data", json.dumps(res))
        logging.debug("result of size {} written to redis at timestamp {}".format(
            len(res), time.time()))
        time.sleep(delay)
