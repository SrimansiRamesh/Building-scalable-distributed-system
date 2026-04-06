"""
Load test for Assignment 10 — Distributed KV Store
Fixed stale read tracking — uses server-returned version numbers
"""

import os
import json
import random
import threading
from locust import HttpUser, task, between, events

KEY_POOL = [f"key-{i:04d}" for i in range(100)]
WRITE_RATIO = float(os.getenv("WRITE_RATIO", "0.5"))
CONFIG = os.getenv("CONFIG", "W5R1")

# Thread-safe version tracker
# Stores the last WRITTEN version per key (from server response)
_version_lock = threading.Lock()
_known_versions = {}   # key -> last version written (from GET after write)
_stale_read_count = 0
_total_read_count = 0


def record_write(key: str, version: int):
    """Record the version we just wrote for this key."""
    with _version_lock:
        _known_versions[key] = version


def check_staleness(key: str, read_version: int) -> bool:
    """
    Returns True if the read version is stale.
    Stale = we know a newer version exists but read returned an older one.
    """
    with _version_lock:
        global _stale_read_count, _total_read_count
        _total_read_count += 1
        known = _known_versions.get(key)
        if known is not None and read_version < known:
            _stale_read_count += 1
            return True
        return False


class KVUser(HttpUser):
    wait_time = between(0.05, 0.2)

    @task
    def do_operation(self):
        key = random.choice(KEY_POOL)
        if random.random() < WRITE_RATIO:
            self._do_write(key)
        else:
            self._do_read(key)

    def _do_write(self, key: str):
        """
        Write a value then immediately read it back to get the
        server-assigned version number. This is the fix — we record
        the server version not a client timestamp.
        """
        value = f"val-{key}-{random.randint(1000, 9999)}"

        with self.client.put(
            f"/kv/{key}",
            json={"value": value},
            catch_response=True,
            name=f"[{CONFIG}] PUT /kv/key"
        ) as resp:
            if resp.status_code == 201:
                resp.success()
                # Read back immediately to get the server version number
                # Use /local/ on the leader to get version without coordination
                r = self.client.get(
                    f"/local/{key}",
                    name=f"[{CONFIG}] local read after write"
                )
                if r.status_code == 200:
                    try:
                        data = r.json()
                        version = data.get("version", 0)
                        if version > 0:
                            record_write(key, version)
                    except Exception:
                        pass
            else:
                resp.failure(f"Write failed: {resp.status_code}")

    def _do_read(self, key: str):
        with self.client.get(
            f"/kv/{key}",
            catch_response=True,
            name=f"[{CONFIG}] GET /kv/key"
        ) as resp:
            if resp.status_code == 200:
                try:
                    data = resp.json()
                    version = data.get("version", 0)
                    is_stale = check_staleness(key, version)
                    if is_stale:
                        # Stale read detected — log but don't fail the request
                        # Stale reads are expected behavior, not errors
                        pass
                    resp.success()
                except Exception:
                    resp.failure("Could not parse response")
            elif resp.status_code == 404:
                # Key not written yet — not a failure
                resp.success()
            else:
                resp.failure(f"Read failed: {resp.status_code}")


@events.quitting.add_listener
def on_quitting(environment, **kwargs):
    with _version_lock:
        total = _total_read_count
        stale = _stale_read_count
        rate = (stale / total * 100) if total > 0 else 0
        print(f"\n{'='*60}")
        print(f"CONFIG: {CONFIG} | WRITE_RATIO: {WRITE_RATIO}")
        print(f"Total reads:  {total}")
        print(f"Stale reads:  {stale} ({rate:.2f}%)")
        print(f"{'='*60}\n")


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    print(f"\nStarting load test: CONFIG={CONFIG} WRITE_RATIO={WRITE_RATIO}")
    print(f"Key pool size: {len(KEY_POOL)} keys")
    print(f"Read/Write split: {(1-WRITE_RATIO)*100:.0f}% reads / {WRITE_RATIO*100:.0f}% writes\n")