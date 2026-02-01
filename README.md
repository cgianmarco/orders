# Orders

## Usage

Have Docker and Docker Compose installed on your machine.

To run the tests, use the following command:

```sh
./scripts/test.sh
```

To run the application, use the following command:

```sh
./scripts/run.sh
```

This will start the application using Docker Compose.

## API Endpoints

- `GET /products`: Retrieve a paginated list of products. Supports cursor-based pagination.
- `POST /orders`: Create a new order with specified items.

