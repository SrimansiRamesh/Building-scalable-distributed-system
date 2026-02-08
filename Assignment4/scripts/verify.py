"""
Verify MapReduce results by comparing against a direct single-pass word count.

Usage:
  # Download the files from S3 first:
  aws s3 cp s3://local-map-reducer/input/sample.txt sample.txt
  aws s3 cp s3://local-map-reducer/results/final_wordcount_XXXXX.json result.json

  # Run verification:
  python scripts/verify.py sample.txt result.json
"""

import sys
import json
import re


def count_words(text):
    """Count words using the same logic as the mapper:
    - Split on whitespace
    - Lowercase
    - Strip non-alpha characters
    - Skip empty strings
    """
    counts = {}
    for word in text.split():
        cleaned = re.sub(r'[^a-zA-Z]', '', word).lower()
        if cleaned:
            counts[cleaned] = counts.get(cleaned, 0) + 1
    return counts


def main():
    if len(sys.argv) < 3:
        print("Usage: python scripts/verify.py <text-file> <mapreduce-result.json>")
        sys.exit(1)

    text_file = sys.argv[1]
    result_file = sys.argv[2]

    # Direct word count
    with open(text_file, "r") as f:
        text = f.read()
    expected = count_words(text)

    # MapReduce result
    with open(result_file, "r") as f:
        actual = json.load(f)

    print(f"Direct count:    {len(expected)} unique words, {sum(expected.values())} total")
    print(f"MapReduce count: {len(actual)} unique words, {sum(actual.values())} total")
    print()

    # Compare
    mismatches = 0

    for word, exp_count in expected.items():
        act_count = actual.get(word, 0)
        if act_count != exp_count:
            print(f"  MISMATCH: '{word}' expected={exp_count} actual={act_count}")
            mismatches += 1

    for word in actual:
        if word not in expected:
            print(f"  EXTRA:    '{word}' count={actual[word]} (not in direct count)")
            mismatches += 1

    print()
    if mismatches == 0:
        print("RESULT: PERFECT MATCH! MapReduce output is correct.")
    else:
        print(f"RESULT: {mismatches} mismatches found.")

    # Show top 10 words
    print()
    print("Top 10 words:")
    sorted_words = sorted(expected.items(), key=lambda x: x[1], reverse=True)
    for word, count in sorted_words[:10]:
        mr_count = actual.get(word, 0)
        match = "✓" if mr_count == count else "✗"
        print(f"  {match} {word:20s} direct={count:5d}  mapreduce={mr_count:5d}")


if __name__ == "__main__":
    main()
