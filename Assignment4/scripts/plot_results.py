"""
Plot actual MapReduce performance results from orchestrator runs.

Usage:
  python scripts/plot_results.py
"""

import matplotlib.pyplot as plt
import numpy as np

# Run 1: 1 mapper, sample.txt (29,579 words)
run_1mapper = {
    "label": "1 Mapper",
    "split_ms": 221.26,
    "map_ms": 445.20,
    "reduce_ms": 189.31,
    "total_ms": 855.77,
    "unique_words": 4797,
    "total_words": 29579,
}

# Run 2: 3 mappers, sample.txt (29,579 words)
run_3mapper = {
    "label": "3 Mappers",
    "split_ms": 267.53,
    "map_ms": 224.31,
    "reduce_ms": 191.06,
    "total_ms": 682.90,
    "unique_words": 4797,
    "total_words": 29579,
}

phases = ["Split", "Map", "Reduce"]
colors = {"Split": "#4CAF50", "Map": "#FF9800", "Reduce": "#F44336"}

fig, axes = plt.subplots(2, 2, figsize=(14, 10))
fig.suptitle("MapReduce Word Count — Performance Results\n(29,579 words | 4,797 unique | 3 chunks)",
             fontsize=15, fontweight="bold")

# ─── Plot 1: Grouped bar chart — 1 mapper vs 3 mappers per phase ───
ax1 = axes[0][0]
x = np.arange(len(phases))
width = 0.3

vals_1 = [run_1mapper["split_ms"], run_1mapper["map_ms"], run_1mapper["reduce_ms"]]
vals_3 = [run_3mapper["split_ms"], run_3mapper["map_ms"], run_3mapper["reduce_ms"]]

bars1 = ax1.bar(x - width/2, vals_1, width, label="1 Mapper", color="#5C6BC0", edgecolor="white")
bars3 = ax1.bar(x + width/2, vals_3, width, label="3 Mappers", color="#26A69A", edgecolor="white")

ax1.set_xticks(x)
ax1.set_xticklabels(phases, fontsize=12)
ax1.set_ylabel("Time (ms)", fontsize=11)
ax1.set_title("Phase Duration: 1 Mapper vs 3 Mappers", fontsize=12, fontweight="bold")
ax1.legend(fontsize=11)
ax1.grid(True, alpha=0.3, axis="y")

# Add value labels on bars
for bar in bars1:
    ax1.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 5,
             f'{bar.get_height():.0f}', ha='center', va='bottom', fontsize=10, fontweight='bold')
for bar in bars3:
    ax1.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 5,
             f'{bar.get_height():.0f}', ha='center', va='bottom', fontsize=10, fontweight='bold')

# ─── Plot 2: Stacked bar — total time breakdown ───
ax2 = axes[0][1]
configs = ["1 Mapper\n(856ms total)", "3 Mappers\n(683ms total)"]
split_vals = [run_1mapper["split_ms"], run_3mapper["split_ms"]]
map_vals = [run_1mapper["map_ms"], run_3mapper["map_ms"]]
reduce_vals = [run_1mapper["reduce_ms"], run_3mapper["reduce_ms"]]

ax2.bar(configs, split_vals, label="Split", color=colors["Split"])
ax2.bar(configs, map_vals, bottom=split_vals, label="Map", color=colors["Map"])
ax2.bar(configs, reduce_vals,
        bottom=[s + m for s, m in zip(split_vals, map_vals)],
        label="Reduce", color=colors["Reduce"])

ax2.set_ylabel("Time (ms)", fontsize=11)
ax2.set_title("Total Pipeline Time Breakdown", fontsize=12, fontweight="bold")
ax2.legend(fontsize=11)
ax2.grid(True, alpha=0.3, axis="y")

# ─── Plot 3: Speedup chart ───
ax3 = axes[1][0]
speedups = [
    run_1mapper["split_ms"] / run_3mapper["split_ms"],
    run_1mapper["map_ms"] / run_3mapper["map_ms"],
    run_1mapper["reduce_ms"] / run_3mapper["reduce_ms"],
    run_1mapper["total_ms"] / run_3mapper["total_ms"],
]
labels = ["Split", "Map", "Reduce", "Total"]
bar_colors = [colors["Split"], colors["Map"], colors["Reduce"], "#2196F3"]

bars = ax3.bar(labels, speedups, color=bar_colors, edgecolor="white", linewidth=1.5)
ax3.axhline(y=1.0, color="gray", linestyle="--", alpha=0.5, label="No speedup (1.0x)")
ax3.set_ylabel("Speedup (×)", fontsize=11)
ax3.set_title("Speedup: 3 Mappers vs 1 Mapper", fontsize=12, fontweight="bold")
ax3.legend(fontsize=10)
ax3.grid(True, alpha=0.3, axis="y")

for bar, val in zip(bars, speedups):
    ax3.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.02,
             f'{val:.2f}×', ha='center', va='bottom', fontsize=12, fontweight='bold')

# ─── Plot 4: Percentage breakdown (side-by-side pie charts) ───
ax4 = axes[1][1]
ax4.axis('off')

# 1 mapper pie
ax4_left = fig.add_axes([0.55, 0.08, 0.18, 0.32])
vals_1_pct = [run_1mapper["split_ms"], run_1mapper["map_ms"], run_1mapper["reduce_ms"]]
ax4_left.pie(vals_1_pct, labels=phases, colors=[colors[p] for p in phases],
             autopct="%1.1f%%", textprops={"fontsize": 9})
ax4_left.set_title("1 Mapper", fontsize=11, fontweight="bold")

# 3 mapper pie
ax4_right = fig.add_axes([0.75, 0.08, 0.18, 0.32])
vals_3_pct = [run_3mapper["split_ms"], run_3mapper["map_ms"], run_3mapper["reduce_ms"]]
ax4_right.pie(vals_3_pct, labels=phases, colors=[colors[p] for p in phases],
              autopct="%1.1f%%", textprops={"fontsize": 9})
ax4_right.set_title("3 Mappers", fontsize=11, fontweight="bold")

plt.savefig("results/performance_results.png", dpi=150, bbox_inches="tight")
print("Saved: performance_results.png")
plt.show()