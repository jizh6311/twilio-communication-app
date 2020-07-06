#!/usr/bin/env bash

test_code() {
	echo ""
	echo "^^^^^^^^^^^^^^^^^^^^^^ $1 ^^^^^^^^^^^^^^^^^^^^^^^^^"
	if [ "$?" = 0 ]; then
		echo "****** command pass****"
	else
		echo "###### command FAIL####"
	fi
	echo ""
}

echo "############################################"
echo "Test Script started..."
echo $(date)

a=$(curl http://localhost:3000/)
test_code grafana_on_3000

a=$(curl http://localhost:9090/)
test_code prometheus_on_9090
curl "http://localhost:9090/api/v1/query?query=up"
echo

a=$(redis-cli -a devpassword ping)
test_code redis_on_6379
redis-cli -a devpassword ping
echo

a=$(curl http://localhost:5000/no_delay)
test_code simulations_on_5000
curl http://localhost:5000/
echo
curl http://localhost:5000/no_delay
echo
curl http://localhost:5000/item_type/dog
echo
curl http://localhost:5000/long_delay
echo

a=$(curl http://localhost:9545/api/v1/configuration)
test_code model_server_on_9545
curl http://localhost:9545/api/v1/configuration
echo
curl http://localhost:9545/api/v1/prototypes
echo
curl http://localhost:9545/api/v1/outlier_scores?n=4
echo
curl http://localhost:9545/api/v1/increment_counter
echo

# TODO  test of model up?

echo "...Test Script complete."
echo "############################################"
