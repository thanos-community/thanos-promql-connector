# thanos-promql-connector
This tool connects to a Prometheus query backend and exposes a grpc server that can be queried by [Thanos Querier](https://github.com/thanos-io/thanos).

This tool currently only supports Google Cloud Monitoring.

## Run thanos-promql-connector
Set up the following environment variables:

```bash
PROJECT_ID=YOUR_PROJECT_ID
# Google Managed Prometheus endpoint.
QUERY_TARGET="https://monitoring.googleapis.com/v1/projects/$PROJECT_ID/location/global/prometheus"
SERVICE_ACCOUNT_CREDENTIAL_FILE=GOOGLE_CLOUD_SERVICE_ACCOUNT_WITH_GOOGLE_CLOUD_MONITORING_READ_ACCESS
```

Run the following command:

```bash
go run main.go \
    --query.target-url=QUERY_TARGET \
    --query.credentials-file=SERVICE_ACCOUNT_CREDENTIAL_FILE
```

## Run with Thanos Querier

1. Start up Thanos Querier by running the following commands:
```bash
git clone https://github.com/thanos-io/thanos.git
cd thanos
go run ./cmd/thanos query --endpoint :8081 --query.mode distributed --query.promql-engine thanos
```

2. Start up the thanos-promql-connector.
3. View Thanos Querier UI at address [0.0.0.0:10902](http://localhost:10902/).