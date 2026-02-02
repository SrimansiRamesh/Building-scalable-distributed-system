from locust import HttpUser, task, between

class AlbumUser(HttpUser):
    # Wait 1-2 seconds between tasks (simulates real user)
    wait_time = between(1, 2)

    @task(3)  # Higher weight - runs more often
    def get_all_albums(self):
        """GET /albums - fetch all albums"""
        self.client.get("/albums")

    @task(2)
    def get_album_by_id(self):
        """GET /albums/:id - fetch single album"""
        self.client.get("/albums/1")

    @task(1)  # Lower weight - runs less often
    def post_album(self):
        """POST /albums - create new album"""
        self.client.post("/albums", json={
            "id": "99",
            "title": "Test Album",
            "artist": "Test Artist",
            "price": 19.99
        })