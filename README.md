# thanos-promql-connector
This tool bridges a PromQL HTTP API query backend to Thanos Querier, exposing a gRPC server that can be queried by [Thanos Querier](https://github.com/thanos-io/thanos).

# How to Run

See the `examples` directory for `docker compose` setups.

## Test and Debugging instructions
1. Update the `config.yaml` file with your desired query target.

2. Start the connector:

```bash
go run . \
    --query.config-file=config.yaml
```

3. Launch Thanos Querier:
```bash
git clone https://github.com/thanos-io/thanos.git
cd thanos
go run ./cmd/thanos query --endpoint :8081 --query.mode distributed --query.promql-engine thanos
```

4. View Thanos Querier UI at address [0.0.0.0:10902](http://localhost:10902/).