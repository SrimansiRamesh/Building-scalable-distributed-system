import random
from locust import FastHttpUser, task, constant

SEARCH_QUERIES = [
    "alpha", "beta", "gamma", "delta", "electronics", "books", "home", "sports"
]

class SearchUser(FastHttpUser):
    # No wait time — each user fires requests as fast as the server responds.
    # With a slow recommend service (2-10s per request), goroutines pile up fast.
    wait_time = constant(0)

    @task
    def search(self):
        query = random.choice(SEARCH_QUERIES)
        self.client.get(f"/products/search?q={query}", name="/products/search")
