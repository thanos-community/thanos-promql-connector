# <b>Example:</b> Connecting Thanos Querier to Prometheus and Google Managed Service for Prometheus (GMP)

This example demonstrates how to connect Thanos Querier to two Prometheus instances and one Google Managed Service for Prometheus (GMP) instance. It provides a clear reference for setting up your own Thanos Querier configuration.

## Prerequisites:

- Docker environment.
- Google Cloud Project with GMP enabled.
- Google Cloud Service Account with the following roles:
    - Monitoring Viewer
    - Metric Writer
- The service account JSON key file downloaded.

## Run Instructions:
1. <b>Edit</b> docker-compose.yml:
    - Insert your `<PROJECT_ID>` on line 44.
    - Replace `<SERVICE_ACCOUNT_CREDENTIAL_FILE_PATH>` on line 47 with the path to your service account file.
2. <b>Start the services:</b> Run the following command:
    ```bash
    docker compose up
    ```
3. <b>Access the UI:</b> Open the Thanos Querier UI in your web browser at http://localhost:10902/. You should be able to query metrics from both your Prometheus instances and your GMP instance.