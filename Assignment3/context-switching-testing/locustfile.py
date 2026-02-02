from locust import task, between
from locust.contrib.fasthttp import FastHttpUser  # Changed import

class AlbumUser(FastHttpUser):  # Changed from HttpUser
    wait_time = between(1, 2)

    @task(3)
    def get_all_albums(self):
        self.client.get("/albums")

    @task(1)
    def post_album(self):
        self.client.post("/albums", json={
            "id": "99",
            "title": "Test Album",
            "artist": "Test Artist",
            "price": 19.99
        })