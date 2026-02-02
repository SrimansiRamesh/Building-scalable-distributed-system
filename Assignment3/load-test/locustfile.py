from locust import HttpUser, task, between

class AlbumUser(HttpUser):
    wait_time = between(1, 2)

    @task(3)  # GET runs 3x more often (3:1 ratio)
    def get_all_albums(self):
        self.client.get("/albums")

    @task(1)  # POST runs 1x
    def post_album(self):
        self.client.post("/albums", json={
            "id": "99",
            "title": "Test Album",
            "artist": "Test Artist",
            "price": 19.99
        })