"""
Part 3: Eventual Consistency Investigation
==========================================
Tests three scenarios against the DynamoDB-backed shopping cart API:

  Scenario 1 — Read-after-write:
    Create cart → GET immediately → record if cart is visible

  Scenario 2 — Item visibility after write:
    Add item → GET cart immediately → record if item appears in items[]

  Scenario 3 — Concurrent updates to same cart:
    10 threads POST different products to the same cart simultaneously
    → GET cart → record how many items are visible

Each scenario retries the read (with timestamps) until consistent or timeout,
so you can measure time-to-consistency.

Usage:
  HOST=http://<alb-or-ip>:8080 python consistency_test.py

Output:
  results/consistency_test_results.json
"""

import json
import os
import random
import threading
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone

import requests

HOST = os.environ.get("HOST", "http://localhost:8080")
RESULTS_FILE = os.environ.get(
    "CONSISTENCY_RESULTS_FILE", "results/consistency_test_results.json"
)
RETRY_INTERVAL_MS = 10   # ms between consistency retry reads
MAX_WAIT_MS = 2000       # give up after 2 seconds
SCENARIO1_RUNS = 20
SCENARIO2_RUNS = 20
SCENARIO3_RUNS = 10
CONCURRENT_WRITERS = 10  # threads for scenario 3


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def create_cart(customer_id: int = 1) -> dict | None:
    """POST /shopping-carts — returns parsed JSON or None on failure."""
    resp = requests.post(f"{HOST}/shopping-carts", json={"customer_id": customer_id}, timeout=5)
    if resp.status_code == 201:
        return resp.json()
    return None


def get_cart(cart_id: str) -> dict | None:
    """GET /shopping-carts/{id} — returns parsed JSON or None on failure."""
    resp = requests.get(f"{HOST}/shopping-carts/{cart_id}", timeout=5)
    if resp.status_code == 200:
        return resp.json()
    return None


def add_item(cart_id: str, product_id: int, quantity: int = 1) -> int:
    """POST /shopping-carts/{id}/items — returns HTTP status code."""
    resp = requests.post(
        f"{HOST}/shopping-carts/{cart_id}/items",
        json={"product_id": product_id, "quantity": quantity},
        timeout=5,
    )
    return resp.status_code


# ---------------------------------------------------------------------------
# Scenario 1: Read-after-write (cart creation)
# ---------------------------------------------------------------------------

def run_scenario1() -> list[dict]:
    """
    Create a cart then immediately GET it.
    Retry until the cart is visible or MAX_WAIT_MS elapses.
    Records whether the first read was consistent and time-to-consistency.
    """
    print(f"\nScenario 1: Read-after-write ({SCENARIO1_RUNS} runs)")
    results = []

    for i in range(SCENARIO1_RUNS):
        created = create_cart(customer_id=random.randint(1, 9999))
        if created is None:
            print(f"  [{i+1}] cart creation failed — skipping")
            continue

        cart_id = created["shopping_cart_id"]
        write_time = time.perf_counter()
        consistent_on_first_read = False
        time_to_consistent_ms = None
        attempts = 0

        while True:
            attempts += 1
            read_start = time.perf_counter()
            cart = get_cart(cart_id)
            elapsed_ms = (time.perf_counter() - write_time) * 1000

            if cart is not None and cart.get("shopping_cart_id") == cart_id:
                time_to_consistent_ms = elapsed_ms
                consistent_on_first_read = attempts == 1
                break

            if elapsed_ms >= MAX_WAIT_MS:
                time_to_consistent_ms = None  # never became consistent
                break

            time.sleep(RETRY_INTERVAL_MS / 1000)

        result = {
            "scenario": "read_after_write",
            "cart_id": cart_id,
            "consistent_on_first_read": consistent_on_first_read,
            "time_to_consistent_ms": time_to_consistent_ms,
            "read_attempts": attempts,
            "timestamp": now_iso(),
        }
        results.append(result)

        status = "CONSISTENT" if consistent_on_first_read else f"DELAYED ({time_to_consistent_ms:.1f}ms)"
        print(f"  [{i+1}] cart={cart_id[:8]}... → {status} (attempts={attempts})")

    return results


# ---------------------------------------------------------------------------
# Scenario 2: Item visibility after write
# ---------------------------------------------------------------------------

def run_scenario2() -> list[dict]:
    """
    Add an item to an existing cart, then immediately GET the cart.
    Retry until the item is visible or MAX_WAIT_MS elapses.
    """
    print(f"\nScenario 2: Item visibility after write ({SCENARIO2_RUNS} runs)")
    results = []

    # Create one cart per run so each test is independent
    for i in range(SCENARIO2_RUNS):
        created = create_cart(customer_id=1)
        if created is None:
            print(f"  [{i+1}] cart creation failed — skipping")
            continue

        cart_id = created["shopping_cart_id"]
        product_id = random.randint(1, 5)
        quantity = random.randint(1, 10)

        # Give the cart a moment to be consistent before the item write
        time.sleep(0.05)

        status_code = add_item(cart_id, product_id, quantity)
        if status_code != 204:
            print(f"  [{i+1}] add_item failed (status={status_code}) — skipping")
            continue

        write_time = time.perf_counter()
        consistent_on_first_read = False
        time_to_consistent_ms = None
        attempts = 0

        while True:
            attempts += 1
            cart = get_cart(cart_id)
            elapsed_ms = (time.perf_counter() - write_time) * 1000

            item_visible = (
                cart is not None
                and any(item["product_id"] == product_id for item in cart.get("items", []))
            )

            if item_visible:
                time_to_consistent_ms = elapsed_ms
                consistent_on_first_read = attempts == 1
                break

            if elapsed_ms >= MAX_WAIT_MS:
                time_to_consistent_ms = None
                break

            time.sleep(RETRY_INTERVAL_MS / 1000)

        result = {
            "scenario": "item_visibility",
            "cart_id": cart_id,
            "product_id": product_id,
            "consistent_on_first_read": consistent_on_first_read,
            "time_to_consistent_ms": time_to_consistent_ms,
            "read_attempts": attempts,
            "timestamp": now_iso(),
        }
        results.append(result)

        status = "CONSISTENT" if consistent_on_first_read else f"DELAYED ({time_to_consistent_ms:.1f}ms)"
        print(f"  [{i+1}] cart={cart_id[:8]}... product={product_id} → {status} (attempts={attempts})")

    return results


# ---------------------------------------------------------------------------
# Scenario 3: Concurrent updates from multiple clients
# ---------------------------------------------------------------------------

def run_scenario3() -> list[dict]:
    """
    10 threads simultaneously POST different products to the same cart.
    After all writes complete, GET the cart and count how many items are visible.
    Retry reads until all CONCURRENT_WRITERS items appear or timeout.
    """
    print(f"\nScenario 3: Concurrent updates ({SCENARIO3_RUNS} runs, {CONCURRENT_WRITERS} writers each)")
    results = []

    for i in range(SCENARIO3_RUNS):
        created = create_cart(customer_id=1)
        if created is None:
            print(f"  [{i+1}] cart creation failed — skipping")
            continue

        cart_id = created["shopping_cart_id"]
        time.sleep(0.1)  # let cart creation settle

        # Each thread writes a distinct product (1..CONCURRENT_WRITERS)
        product_ids = list(range(1, min(CONCURRENT_WRITERS, 5) + 1))  # max 5 (seed products)
        write_errors = []

        def write_item(pid: int):
            sc = add_item(cart_id, pid, quantity=1)
            if sc != 204:
                write_errors.append(pid)

        with ThreadPoolExecutor(max_workers=len(product_ids)) as ex:
            futures = [ex.submit(write_item, pid) for pid in product_ids]
            for f in as_completed(futures):
                f.result()

        expected_count = len(product_ids) - len(write_errors)
        write_time = time.perf_counter()
        final_visible = 0
        time_to_consistent_ms = None
        attempts = 0

        while True:
            attempts += 1
            cart = get_cart(cart_id)
            elapsed_ms = (time.perf_counter() - write_time) * 1000
            visible = len(cart.get("items", [])) if cart else 0

            if visible >= expected_count:
                final_visible = visible
                time_to_consistent_ms = elapsed_ms
                break

            if elapsed_ms >= MAX_WAIT_MS:
                final_visible = visible
                break

            time.sleep(RETRY_INTERVAL_MS / 1000)

        all_visible = final_visible >= expected_count
        result = {
            "scenario": "concurrent_updates",
            "cart_id": cart_id,
            "writers": len(product_ids),
            "write_errors": len(write_errors),
            "expected_items": expected_count,
            "visible_items_after_read": final_visible,
            "all_items_visible": all_visible,
            "time_to_consistent_ms": time_to_consistent_ms,
            "read_attempts": attempts,
            "timestamp": now_iso(),
        }
        results.append(result)

        status = f"ALL VISIBLE" if all_visible else f"PARTIAL ({final_visible}/{expected_count})"
        print(f"  [{i+1}] cart={cart_id[:8]}... → {status} in {time_to_consistent_ms:.1f}ms (attempts={attempts})")

    return results


# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

def print_summary(all_results: list[dict]) -> None:
    print("\n" + "=" * 60)
    print("CONSISTENCY SUMMARY")
    print("=" * 60)

    for scenario_key, label in [
        ("read_after_write", "Scenario 1: Read-after-write"),
        ("item_visibility",  "Scenario 2: Item visibility"),
        ("concurrent_updates", "Scenario 3: Concurrent updates"),
    ]:
        subset = [r for r in all_results if r["scenario"] == scenario_key]
        if not subset:
            continue

        if scenario_key in ("read_after_write", "item_visibility"):
            first_read_consistent = sum(1 for r in subset if r["consistent_on_first_read"])
            times = [r["time_to_consistent_ms"] for r in subset if r["time_to_consistent_ms"] is not None]
            avg_ms = sum(times) / len(times) if times else 0
            never = sum(1 for r in subset if r["time_to_consistent_ms"] is None)
            print(f"\n{label} ({len(subset)} runs):")
            print(f"  Consistent on first read : {first_read_consistent}/{len(subset)}")
            print(f"  Avg time-to-consistent   : {avg_ms:.2f} ms")
            print(f"  Never became consistent  : {never}")
        else:
            all_vis = sum(1 for r in subset if r["all_items_visible"])
            times = [r["time_to_consistent_ms"] for r in subset if r["time_to_consistent_ms"] is not None]
            avg_ms = sum(times) / len(times) if times else 0
            print(f"\n{label} ({len(subset)} runs):")
            print(f"  All items visible        : {all_vis}/{len(subset)}")
            print(f"  Avg time-to-consistent   : {avg_ms:.2f} ms")

    print("=" * 60)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    print(f"Consistency test against: {HOST}")
    print(f"Results file: {RESULTS_FILE}")

    all_results = []
    all_results.extend(run_scenario1())
    all_results.extend(run_scenario2())
    all_results.extend(run_scenario3())

    print_summary(all_results)

    os.makedirs(os.path.dirname(RESULTS_FILE) if os.path.dirname(RESULTS_FILE) else ".", exist_ok=True)
    with open(RESULTS_FILE, "w", encoding="utf-8") as f:
        json.dump(all_results, f, indent=2)

    print(f"\nResults written to {RESULTS_FILE}")
