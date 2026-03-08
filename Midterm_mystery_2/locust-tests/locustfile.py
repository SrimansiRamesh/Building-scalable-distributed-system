
import random
from locust import FastHttpUser, task, constant_throughput

# Common search terms that match generated product data
SEARCH_QUERIES = [
    # Brands
    "alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta",
    # Categories
    "electronics", "books", "home", "garden", "sports", "toys", "clothing", "food",
    # Common term that matches ALL products
    "product",
]


class SearchUser(FastHttpUser):
    """
    Simulates users hammering the search endpoint.
    Uses FastHttpUser for minimal client-side overhead.
    Minimal wait time to maximize CPU pressure on the server.
    """
    # Minimal wait — each user sends as many requests as possible
    wait_time = constant_throughput(10)  # 10 requests per second per user

    @task
    def search_product(self):
        """Search with a random common term."""
        query = random.choice(SEARCH_QUERIES)
        self.client.get(f"/products/search?q={query}", name="/products/search")

