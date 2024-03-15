# <b>Example:</b> Connecting Thanos Querier to Prometheus and Google Managed Service for Prometheus (GMP)

This example demonstrates how to connect Thanos Querier to two Prometheus instances and one Google Managed Service for Prometheus (GMP) instance. It provides a clear reference for setting up your own Thanos Querier configuration.

## Prerequisites:

- Docker environment.
- Google Cloud Service Account with the following roles:
    - Monitoring Viewer
- The service account JSON key file downloaded.

## Run Instructions:
1. <b>Edit</b> docker-compose.yml:
    - Insert your `<PROJECT_ID>` in the config.yaml
    - Add your service account JSON key file to this folder and name it `key.json`.
2. <b>Start the services:</b> Run the following command:
    ```bash
    docker compose up
    ```
3. <b>Access the UI:</b> Open the Thanos Querier UI in your web browser at http://localhost:10902/. You should be able to query metrics from both the Prometheus instances and your GMP instance.