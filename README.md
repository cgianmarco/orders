# Orders

## Usage

Have Docker and Docker Compose installed on your machine.

To run the tests, use the following command:

```sh
./scripts/tests.sh
```

To run the application, use the following command:

```sh
./scripts/run.sh
```

This will start the application using Docker Compose.

## API Endpoints

### GET /products
Retrieve a paginated list of products (cursor-based).

**Example**
```
GET /products?limit=5
```

**Response notes**
- Returns `nextCursor`.
- Use it like:
```
GET /products?limit=2&cursor=<cursor>
```

### POST /orders
Create a new order with specified items.

**Request body**
```json
{
    "items": [
        {
            "id": 2,
            "quantity": 2
        },
        {
            "id": 3,
            "quantity": 3
        }
    ]
}
```

**Example response**
```json
{
    "id": 2,
    "totalPriceCents": 27600,
    "totalVATCents": 6072,
    "items": [
        {
            "id": 2,
            "priceCents": 2550,
            "vatCents": 561,
            "quantity": 2
        },
        {
            "id": 3,
            "priceCents": 7500,
            "vatCents": 1650,
            "quantity": 3
        }
    ]
}
```
