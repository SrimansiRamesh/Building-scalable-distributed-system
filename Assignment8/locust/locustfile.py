import logging
import json
import os
import threading
import time
import random
from datetime import datetime, timezone

from locust import FastHttpUser, events, task
from locust.exception import StopUser

# Global result storage (each user appends; persist once on quit)
_lock = threading.Lock()
_results = []

MAX_PER_OPERATION = 50
RESULTS_FILE = os.environ.get("RESULTS_FILE", "mysql_test_results.json")

logger = logging.getLogger(__name__)


def _record_result(operation: str, response, start_time: float) -> None:
    """Record a single operation result."""
    end_time = time.perf_counter()
    response_time_ms = (end_time - start_time) * 1000.0
    timestamp = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")

    status_code = response.status_code if response is not None else None
    if operation in ("create_cart", "add_items"):
        success = status_code in (200, 201, 204)
    else:
        success = status_code == 200

    entry = {
        "operation": operation,
        "response_time": response_time_ms,
        "success": success,
        "status_code": status_code,
        "timestamp": timestamp,
    }

    with _lock:
        _results.append(entry)

    logger.info(
        "locust op=%s status_code=%s success=%s",
        operation,
        status_code,
        success
    )


class ShoppingCartUser(FastHttpUser):
    """
    Locust user that runs exactly 150 shopping cart operations per spawned user:
    - 50 POST /shopping-carts
    - 50 POST /shopping-carts/{id}/items
    - 50 GET  /shopping-carts/{id}

    The server endpoint is provided via Locust's standard `--host` option.
    """

    def on_start(self) -> None:
        self._create_count = 0
        self._add_count = 0
        self._get_count = 0
        self._cart_ids: list = []

    @task
    def shopping_cart_flow(self) -> None:
        # Decide which operation to run next to satisfy the exact counts for this user.
        if self._create_count < MAX_PER_OPERATION:
            op = "create_cart"
        elif self._add_count < MAX_PER_OPERATION:
            op = "add_items"
        elif self._get_count < MAX_PER_OPERATION:
            op = "get_cart"
        else:
            raise StopUser()

        if op == "create_cart":
            self._do_create_cart()
        elif op == "add_items":
            self._do_add_items()
        else:
            self._do_get_cart()

    def _do_create_cart(self) -> None:
        payload = {"customer_id": 1}
        start = time.perf_counter()
        try:
            resp = self.client.post("/shopping-carts", json=payload, name="POST /shopping-carts")
        except Exception:
            resp = None
        _record_result("create_cart", resp, start)

        self._create_count += 1
        if resp is not None and resp.ok:
            try:
                data = resp.json()
                cart_id = data.get("shopping_cart_id")
                if cart_id is not None:
                    self._cart_ids.append(cart_id)
            except Exception:
                pass

        if self._user_done():
            raise StopUser()

    def _do_add_items(self) -> None:
        if not self._cart_ids:
            if self._create_count < MAX_PER_OPERATION:
                self._do_create_cart()
            else:
                raise StopUser()
            return

        cart_id = random.choice(self._cart_ids)

        payload = {
            "product_id": random.randint(1, 5),
            "quantity": random.randint(1, 20),
        }
        path = f"/shopping-carts/{cart_id}/items"
        start = time.perf_counter()
        try:
            resp = self.client.post(path, json=payload, name="POST /shopping-carts/{id}/items")
        except Exception:
            resp = None
        _record_result("add_items", resp, start)

        self._add_count += 1
        if self._user_done():
            raise StopUser()

    def _do_get_cart(self) -> None:
        if not self._cart_ids:
            if self._create_count < MAX_PER_OPERATION:
                self._do_create_cart()
            else:
                raise StopUser()
            return

        cart_id = self._cart_ids[0]

        path = f"/shopping-carts/{cart_id}"
        start = time.perf_counter()
        try:
            resp = self.client.get(path, name="GET /shopping-carts/{id}")
        except Exception:
            resp = None
        _record_result("get_cart", resp, start)

        self._get_count += 1
        if self._user_done():
            raise StopUser()

    def _user_done(self) -> bool:
        return (
            self._create_count >= MAX_PER_OPERATION
            and self._add_count >= MAX_PER_OPERATION
            and self._get_count >= MAX_PER_OPERATION
        )


@events.quitting.add_listener
def _persist_results_on_quit(environment, **kwargs) -> None:
    """Write aggregated results when the run ends (any reason)."""
    _write_results_file()


def _write_results_file() -> None:
    if getattr(_write_results_file, "_written", False):
        return
    setattr(_write_results_file, "_written", True)
    try:
        logger.info("writing %d results to %s", len(_results), RESULTS_FILE)
        with open(RESULTS_FILE, "w", encoding="utf-8") as f:
            json.dump(_results, f, indent=2)
    except Exception:
        logger.exception("failed writing results to %s", RESULTS_FILE)