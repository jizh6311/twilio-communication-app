import numpy as np
import unittest

from amp_anomaly_models.prometheus_model import *


class MyTestCase(unittest.TestCase):

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



if __name__ == '__main__':
    unittest.main()
