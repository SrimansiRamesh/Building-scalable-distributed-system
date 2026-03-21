#!/usr/bin/env python3
"""
Build a dot plot (strip plot with jitter) of per-request response times by operation.

Reads Locust JSON exports from results/ (array of {operation, response_time, ...}).

Usage (from locust/):
  python plot_response_times.py
  python plot_response_times.py --input results/mysql_test_results.json --output results/my_plot.png
"""

from __future__ import annotations

import argparse
import json
import random
from collections import defaultdict
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np


def load_records(path: Path) -> list[dict]:
    with path.open(encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, list):
        raise ValueError(f"Expected JSON array in {path}")
    return data


def main() -> None:
    parser = argparse.ArgumentParser(description="Dot plot of response times by operation.")
    here = Path(__file__).resolve().parent
    parser.add_argument(
        "--input",
        type=Path,
        default=here / "results" / "mysql_test_results.json",
        help="JSON file with Locust per-request records",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=here / "results" / "response_time_dotplot.png",
        help="Where to write the PNG",
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=42,
        help="RNG seed for horizontal jitter (reproducible layout)",
    )
    args = parser.parse_args()

    random.seed(args.seed)
    np.random.seed(args.seed)

    records = load_records(args.input)
    by_op: dict[str, list[float]] = defaultdict(list)
    success_by_op: dict[str, list[bool]] = defaultdict(list)

    for row in records:
        op = row.get("operation")
        rt = row.get("response_time")
        if op is None or rt is None:
            continue
        by_op[str(op)].append(float(rt))
        success_by_op[str(op)].append(bool(row.get("success", True)))

    if not by_op:
        raise SystemExit("No valid records with operation and response_time")

    # Stable column order: workflow order when all three exist, else sorted
    preferred = ["create_cart", "add_items", "get_cart"]
    ops = [o for o in preferred if o in by_op]
    ops.extend(sorted(o for o in by_op if o not in preferred))

    fig, ax = plt.subplots(figsize=(9, 5))
    jitter_hw = 0.12
    legend_ok = {"success": False, "non-200": False}

    for idx, op in enumerate(ops):
        times = np.asarray(by_op[op], dtype=float)
        ok = np.asarray(success_by_op[op], dtype=bool)
        n = len(times)
        base_x = np.full(n, idx, dtype=float)
        jitter = (np.random.random(n) - 0.5) * 2 * jitter_hw
        x = base_x + jitter

        fail = ~ok
        if ok.any() and fail.any():
            if not legend_ok["success"]:
                ax.scatter(x[ok], times[ok], s=14, alpha=0.65, c="#2e7d32", edgecolors="none", label="success")
                legend_ok["success"] = True
            else:
                ax.scatter(x[ok], times[ok], s=14, alpha=0.65, c="#2e7d32", edgecolors="none")
            if not legend_ok["non-200"]:
                ax.scatter(x[fail], times[fail], s=22, alpha=0.85, c="#c62828", marker="x", linewidths=1.2, label="non-200 (Locust)")
                legend_ok["non-200"] = True
            else:
                ax.scatter(x[fail], times[fail], s=22, alpha=0.85, c="#c62828", marker="x", linewidths=1.2)
        elif fail.any():
            lbl = "non-200 (Locust)" if not legend_ok["non-200"] else None
            ax.scatter(x[fail], times[fail], s=22, alpha=0.85, c="#c62828", marker="x", linewidths=1.2, label=lbl)
            legend_ok["non-200"] = True
        else:
            ax.scatter(x, times, s=14, alpha=0.65, c="#1565c0", edgecolors="none")

    ax.set_xticks(range(len(ops)))
    ax.set_xticklabels(ops, rotation=15, ha="right")
    ax.set_ylabel("Response time (ms)")
    ax.set_xlabel("Operation")
    ax.set_title("Per-request response time by operation")
    ax.grid(True, axis="y", alpha=0.35)
    handles, labels = ax.get_legend_handles_labels()
    if handles:
        ax.legend(loc="upper right", framealpha=0.9)

    fig.tight_layout()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    fig.savefig(args.output, dpi=150, bbox_inches="tight")
    plt.close(fig)
    print(f"Wrote {args.output}")


if __name__ == "__main__":
    main()
