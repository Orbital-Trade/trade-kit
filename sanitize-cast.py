#!/usr/bin/env python3
"""sanitize-cast.py — Strip sensitive data from asciinema .cast files.

Usage:
    python3 sanitize-cast.py input.cast output.cast
    python3 sanitize-cast.py input.cast output.cast --verify

The .cast file is JSON-lines (first line is header, rest are [time, type, text]).
This script does string replacement on sensitive patterns.
"""

import re
import sys

# ── Static replacements (add real values before recording) ───────────────────

REPLACEMENTS = {
    # System paths
    "/home/jramirez": "/home/trader",
    "jramirez": "trader",

    # Tiger account (replace with your real values)
    "50392935": "XXXXXXXX",
    "20158945": "XXXXXXXX",

    # Demo tokens
    "demo_token_12345": "xxxxx",
}

# ── Regex patterns ───────────────────────────────────────────────────────────

REGEX_PATTERNS = [
    # Tiger account numbers (8 digits in "account: NNNNNNNN" context)
    (r'account:\s*\d{8}', 'account: XXXXXXXX'),
    # API keys (ot_xxxx format)
    (r'ot_[a-zA-Z0-9]{10,}', 'ot_xxxxxxxxxxxxx'),
    # RSA private keys
    (r'-----BEGIN[A-Z ]+KEY-----[\s\S]*?-----END[A-Z ]+KEY-----', '-----BEGIN PRIVATE KEY-----\\nREDACTED\\n-----END PRIVATE KEY-----'),
    # Telegram bot tokens
    (r'\d{9,}:[A-Za-z0-9_-]{30,}', 'XXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'),
    # Discord webhook URLs
    (r'https://discord\.com/api/webhooks/\d+/[A-Za-z0-9_-]+', 'https://discord.com/api/webhooks/XXXXX/XXXXX'),
    # Email addresses
    (r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', 'user@example.com'),
    # IP:port combos (except localhost standard ones)
    (r'\b(?!127\.0\.0\.1)\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d+\b', 'X.X.X.X:XXXX'),
    # Trade passwords (6-digit PINs in context)
    (r'TRADE_PASSWORD=\d+', 'TRADE_PASSWORD=XXXXXX'),
    # Private key base64 blobs (long base64 strings)
    (r'PRIVATE_KEY=[A-Za-z0-9+/=]{40,}', 'PRIVATE_KEY=REDACTED'),
    (r'private_key_pk8=[A-Za-z0-9+/=]{40,}', 'private_key_pk8=REDACTED'),
]

# ── Verify patterns (must be absent after sanitization) ──────────────────────

VERIFY_PATTERNS = [
    r'jramirez',
    r'50392935',
    r'20158945',
    r'ot_[a-zA-Z0-9]{10,}',
    r'-----BEGIN.*KEY-----',
    r'\d{9,}:[A-Za-z0-9_-]{30,}',
    r'TRADE_PASSWORD=\d{6}',
]


def sanitize_line(line: str) -> str:
    for old, new in REPLACEMENTS.items():
        line = line.replace(old, new)
    for pattern, replacement in REGEX_PATTERNS:
        line = re.sub(pattern, replacement, line)
    return line


def sanitize_cast(input_path: str, output_path: str) -> None:
    with open(input_path, 'r') as f:
        lines = f.readlines()

    sanitized = [sanitize_line(line) for line in lines]

    with open(output_path, 'w') as f:
        f.writelines(sanitized)

    print(f"Sanitized {len(lines)} lines -> {output_path}")


def verify(path: str) -> bool:
    with open(path, 'r') as f:
        content = f.read()

    clean = True
    for pattern in VERIFY_PATTERNS:
        matches = re.findall(pattern, content)
        if matches:
            print(f"  FAIL: '{pattern}' found {len(matches)}x: {matches[:3]}")
            clean = False

    if clean:
        print("  PASS: no sensitive patterns found")
    return clean


def main():
    if len(sys.argv) < 3:
        print("Usage: python3 sanitize-cast.py input.cast output.cast [--verify]")
        sys.exit(1)

    sanitize_cast(sys.argv[1], sys.argv[2])

    if '--verify' in sys.argv:
        print("\nVerifying...")
        if not verify(sys.argv[2]):
            print("\nWARNING: sensitive patterns remain!")
            sys.exit(1)


if __name__ == '__main__':
    main()
