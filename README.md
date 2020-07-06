## MODEL SIMULATION AND TEST PLATFORM
```
Project
$ tree -L 2
.
├── README.md
├── amp_model
│   ├── Dockerfile-model
│   ├── dependencies
│   ├── models
│   └── requirements.txt
├── amp_model_api
│   ├── Dockerfile-model-api
│   ├── Dockerfile-nginx
│   ├── api
│   ├── config
│   ├── dependencies
│   └── requirements.txt
├── amp_simulated_clients
│   ├── README.md
│   ├── clients
│   ├── requirements.txt
│   └── setup.py
├── amp_simulated_servers
│   ├── Dockerfile-servers
│   ├── requirements.txt
│   └── servers
├── before_docker_compose.bash
├── common_configs
│   └── model_api.yml
├── docker-compose.yml
└── up_test_script.bash
```

### Run It
In the directory "amp_simulation"

    ./before_docker_compose.bash
	docker-compose up --build

or

	docker-compose up

create some traffic by running the simulated clients:

	cd amp_simulated_clients/ && virtualenv .env-clients && source .env-clients/bin/activate && pip install -r requirements.txt && python clients/clients.py

### Endpoints Available

- Prometheus:	http://localhost:9090/graph

- Grafana:	http://localhost:3000

- Mock Server:
    - http://localhost:5000/
    - http://localhost:5000/long_delay
    - http://localhost:5000/long_delay_outlier
    - http://localhost:5000/short_delay
    - http://localhost:5000/short_delay_outlier
    - http://localhost:5000/no_delay
    - http://localhost:5000/item_type/<anythign>
    - http://localhost:5000/status/<int>
    - http://localhost:5000/metrics    <<< Prometheus Scrape

- Model:
    - http://localhost:9545/api/v1/configuration
	- http://localhost:9545/api/v1/prototypes
	- http://localhost:9545/api/v1/outlier_scores
	- http://localhost:9545/api/v1/increment_counter
	- http://localhost:9545/api/v1/query?query=amp_model_http_request_total


### Requirements
Python 3.6+, then,

	pip install virtualenv


