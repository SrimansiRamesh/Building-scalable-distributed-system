"""
Locust load tests for the Product API.

Two user classes:
  - ProductHttpUser     (standard HttpUser)
  - ProductFastHttpUser (FastHttpUser — uses gevent-based HTTP client)

Traffic pattern:
  80/20 read/write ratio (realistic e-commerce traffic).

Run one at a time:
  locust -f locustfile.py --host http://34.229.204.65:8080 ProductHttpUser
  locust -f locustfile.py --host http://34.229.204.65:8080 ProductFastHttpUser

Headless:
  locust -f locustfile.py --host http://34.229.204.65:8080 \
    --headless -u 100 -r 10 --run-time 60s ProductHttpUser
"""

import random
from locust import HttpUser, FastHttpUser, task, between

# -----------------------------------------------------------------
# Shared product pool — tracks which product IDs have been created
# so GET requests target real products most of the time.
# -----------------------------------------------------------------
created_product_ids = set()
next_product_id = 1


def _make_product(pid):
    """Build a valid product payload matching the OpenAPI spec."""
    return {
        "product_id": pid,
        "sku": f"SKU-{pid:05d}",
        "manufacturer": random.choice([
            "Acme Corporation",
            "Globex Inc",
            "Initech",
            "Umbrella Corp",
            "Stark Industries",
            "Wayne Enterprises",
        ]),
        "category_id": random.randint(1, 50),
        "weight": random.randint(100, 5000),
        "some_other_id": random.randint(1, 1000),
    }


def _do_get_product(client):
    """GET a product — called by both user types."""
    global next_product_id
    if created_product_ids:
        if random.random() < 0.7:
            pid = random.choice(list(created_product_ids))
        else:
            pid = random.randint(1, max(next_product_id, 100))
    else:
        pid = random.randint(1, 100)
    client.get(f"/products/{pid}", name="/products/[id]")


def _do_create_product(client):
    """POST a new product — called by both user types."""
    global next_product_id
    pid = next_product_id
    next_product_id += 1
    payload = _make_product(pid)
    with client.post(
        f"/products/{pid}/details",
        json=payload,
        name="/products/[id]/details",
        catch_response=True,
    ) as response:
        if response.status_code == 204:
            created_product_ids.add(pid)
            response.success()
        else:
            response.failure(f"Expected 204, got {response.status_code}")


# -----------------------------------------------------------------
# Standard HttpUser (uses Python requests under the hood)
# -----------------------------------------------------------------
class ProductHttpUser(HttpUser):
    """
    Standard Locust HttpUser.
    Uses Python's `requests` library — easy to use, full featured,
    but has higher per-request overhead due to connection management.
    """
    wait_time = between(0.5, 2)

    @task(4)
    def get_product(self):
        _do_get_product(self.client)

    @task(1)
    def create_product(self):
        _do_create_product(self.client)


# -----------------------------------------------------------------
# FastHttpUser (uses geventhttpclient — much lower overhead)
# -----------------------------------------------------------------
class ProductFastHttpUser(FastHttpUser):
    """
    Locust FastHttpUser.
    Uses geventhttpclient — significantly less CPU overhead per request,
    which means a single Locust worker can generate more load.

    You'll see the difference at HIGH concurrency (500+ users).
    At low concurrency (<50 users), both behave similarly because
    the bottleneck is the server, not the load generator.
    """
    wait_time = between(0.5, 2)

    @task(4)
    def get_product(self):
        _do_get_product(self.client)

    @task(1)
    def create_product(self):
        _do_create_product(self.client)
