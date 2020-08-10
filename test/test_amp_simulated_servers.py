import unittest
import subprocess
import os
import requests
import time

class MyTestCase(unittest.TestCase):

    @classmethod
    def setUpClass(self):
        # set up local, lightweight server environment
        tmp_dir_name = os.path.expanduser("~/deleteme")
        try:
            os.mkdir(tmp_dir_name)
        except FileExistsError as e:
            pass
        os.environ["prometheus_multiproc_dir"] = tmp_dir_name
        self.proc_output = subprocess.Popen(["poetry", "run", "python",
                                             "amp_simulated_servers/flask_server.py"])
        time.sleep(1)

    @classmethod
    def tearDownClass(self):
        time.sleep(1)
        print(self.proc_output)
        self.proc_output.kill()

    def setUp(self):
        self.url = "http://localhost:5000/"

    def test_root_endpoint(self):
        a = requests.get(self.url)
        res = a.json()
        self.assertEqual(res["endpoint"], "/")
        self.assertIn("params", res)

    def test_no_delay_endpoint(self):
        a = requests.get(self.url + "no_delay")
        res = a.json()
        self.assertEqual(res["endpoint"], "no_delay")
        self.assertIn("params", res)

    def test_item_type_endpoint(self):
        a = requests.get(self.url + "item_type/big")
        res = a.json()
        self.assertEqual(res["endpoint"], "item_type")
        self.assertIn("params", res)

    def test_long_delay_endpoint(self):
        a = requests.get(self.url + "long_delay")
        res = a.json()
        self.assertEqual(res["endpoint"], "long_delay")
        self.assertIn("params", res)

    def test_long_delay_outlier_endpoint(self):
        a = requests.get(self.url + "long_delay_outlier")
        res = a.json()
        self.assertEqual(res["endpoint"], "long_delay_outlier")
        self.assertIn("params", res)

    def test_short_delay_endpoint(self):
        a = requests.get(self.url + "short_delay")
        res = a.json()
        self.assertEqual(res["endpoint"], "short_delay")
        self.assertIn("params", res)

    def test_short_delay_outlier_endpoint(self):
        a = requests.get(self.url + "short_delay_outlier")
        res = a.json()
        self.assertEqual(res["endpoint"], "short_delay_outlier")
        self.assertIn("params", res)

    def test_status_endpoint(self):
        a = requests.get(self.url + "status/1234")
        res = a.json()
        self.assertEqual(res["endpoint"], "echo_status")
        self.assertIn("params", res)

    def test_metrics_endpoint(self):
        a = requests.get(self.url + "metrics")
        res = a.text
        print(res)
        self.assertTrue(res.startswith("# HELP"))

if __name__ == '__main__':
    unittest.main()
