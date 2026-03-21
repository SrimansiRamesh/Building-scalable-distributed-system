import random
from locust import HttpUser, task, between

SAMPLE_ORDERS = [
    {
        "customer_id": random.randint(1000, 9999),
        "items": [
            {"product_id": "SKU-001", "quantity": 2, "price": 29.99},
            {"product_id": "SKU-042", "quantity": 1, "price": 59.99},
        ],
    },
    {
        "customer_id": random.randint(1000, 9999),
        "items": [
            {"product_id": "SKU-007", "quantity": 3, "price": 9.99},
        ],
    },
    {
        "customer_id": random.randint(1000, 9999),
        "items": [
            {"product_id": "SKU-100", "quantity": 1, "price": 199.99},
            {"product_id": "SKU-101", "quantity": 2, "price": 14.99},
            {"product_id": "SKU-202", "quantity": 5, "price": 4.99},
        ],
    },
]


def make_order():
    order = random.choice(SAMPLE_ORDERS).copy()
    order["customer_id"] = random.randint(1000, 9999)
    return order


class SyncOrderUser(HttpUser):
    """
    Phase 1 — synchronous endpoint.

    Normal:     locust --headless -u 5  -r 1  -t 30s --host http://<ALB> --class-picker SyncOrderUser
    Flash sale: locust --headless -u 20 -r 10 -t 60s --host http://<ALB> --class-picker SyncOrderUser
    """

    wait_time = between(0.1, 0.5)

    @task
    def place_order_sync(self):
        with self.client.post(
            "/orders/sync",
            json=make_order(),
            name="/orders/sync",
            catch_response=True,
            timeout=30,
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"status {response.status_code}: {response.text[:100]}")


class AsyncOrderUser(HttpUser):
    """
    Phase 2 — async endpoint.

    Flash sale: locust --headless -u 20 -r 10 -t 60s --host http://<ALB> --class-picker AsyncOrderUser

    Expected: ~100% acceptance rate, <100ms response times.
    Payment processing happens in the background via SQS worker.
    """

    wait_time = between(0.1, 0.5)

    @task
    def place_order_async(self):
        with self.client.post(
            "/orders/async",
            json=make_order(),
            name="/orders/async",
            catch_response=True,
            timeout=5,  # should return almost instantly
        ) as response:
            if response.status_code == 202:
                response.success()
            else:
                response.failure(f"status {response.status_code}: {response.text[:100]}")
