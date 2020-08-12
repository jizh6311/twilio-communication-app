import numpy as np
import unittest
import subprocess
import threading
import multiprocessing

from amp_anomaly_models.prometheus_model import *


class MyTestCase(unittest.TestCase):

    @classmethod
    def setUpClass(self):
#         # set up local, lightweight server environment
#         tmp_dir_name = os.path.expanduser("~/deleteme")
#         try:
#             os.mkdir(tmp_dir_name)
#         except FileExistsError as e:
#             pass
#         os.environ["prometheus_multiproc_dir"] = tmp_dir_name
        self.procs = [
            subprocess.Popen(["docker", "run","--publish", "6379:6379",
                              "redis:5.0-alpine", "--requirepass", "devpassword"]),
            subprocess.Popen(["docker", "run", "--publish", "9090:9090", "prom/prometheus"]),
            subprocess.Popen(["poetry", "run", "python", "amp_anomaly_models/prometheus_model.py"])
        ]
        time.sleep(1)

    @classmethod
    def tearDownClass(self):
        for p in self.procs:
            print(p)
            p.kill()

    def setUp(self):
        self.onscs = OutlierNodeSumCountStrategy()
        self.ots = OutlierTypeStrategy()

    def test_outlier_score(self):
        v = None
        w = None
        a = self.onscs.outlier_score(v, w)
        self.assertEqual(a, (0, 0, 0, 0))
        #
        v = np.random.uniform(size=(100,))
        v[0] = 5.5
        print(v)
        a = self.onscs.outlier_score(v, w)
        self.assertEqual(a, (100, 1, 100, 1.0))

    def test_analyze_metric(self):
        r = Metric(
            {
                "metric":
                    {"__name__": "the_metric"},
                "values": list(range(100))
            },
            base_metric__name__="the_name")
        d = [
            {
                "n": 0, "n_out": 0, "uniq": 0, "outlier_score": 0, "w": 1
            }
        ]
        a = {}
        with self.assertLogs(level='DEBUG') as cm:
            self.onscs.analyze_metric(r, d, a)
        self.assertEqual(cm.output, ['DEBUG:root:processing the_metric'])

    def test_metric_filter(self):
        l = [
            {
                "outlier_score": 1,
                "w": 1,
            },
            {
                "outlier_score": 10,
                "w": 10,
            },
        ]
        o = self.onscs.metric_filter(l, 4, 3)
        self.assertEqual(len(o), 1)
        self.assertEqual(o[0]["w"], 10)

    def test_main_loop(self):
        ts = time.time()
        # start the main_loop to load a data record
        p = multiprocessing.Process(target=main_loop)
        p.start()
        time.sleep(3)  # takes a few seconds to get a record in...
        redis_inst = redis.StrictRedis(host="localhost", port=6379, db=0,
                                       decode_responses=True, password="devpassword")
        redis_inst.client_setname("amp_model-outlier_model")
        res = redis_inst.hgetall("outlier_scores")
        print(res)
        self.assertIn("data", res)
        self.assertIn("timestamp", res)
        self.assertGreater(float(res["timestamp"]), ts)
        p.terminate()

if __name__ == '__main__':
    unittest.main()
