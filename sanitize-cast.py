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
import json

# ── Patterns to sanitize ────────────────────────────────────────────────────

REPLACEMENTS = {
    # Account IDs / API keys (add your real values here before running)
    # "REAL_TIGER_ID": "20150000",
    # "REAL_ACCOUNT": "50390000",
    # "real_api_key_here": "etoro_api_key_xxxxx",
    # "real_user_key_here": "etoro_user_key_xxxxx",

    # Common system paths
    "/home/jramirez": "/home/trader",
    "jramirez": "trader",

    # Auth tokens from demo
    "demo_token_12345": "xxxxx_demo_token",

    # IP addresses (private ranges OK, but sanitize just in case)
    "127.0.0.1:19091": "127.0.0.1:19090",
}

# Regex patterns to catch anything the static replacements miss
REGEX_PATTERNS = [
    # API keys (ot_xxxx format)
    (r'ot_[a-zA-Z0-9]{10,}', 'ot_xxxxxxxxxxxxx'),
    # UUIDs (request IDs, etc.)
    # (r'[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}', 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'),
    # RSA private keys (base64 blobs)
    (r'-----BEGIN[A-Z ]+KEY-----[A-Za-z0-9+/=\s]+-----END[A-Z ]+KEY-----', '-----BEGIN PRIVATE KEY-----\nXXXXX_REDACTED\n-----END PRIVATE KEY-----'),
    # Telegram bot tokens
    (r'\d{9,}:[A-Za-z0-9_-]{30,}', 'XXXXXXXXX:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'),
    # Discord webhook URLs
    (r'https://discord\.com/api/webhooks/\d+/[A-Za-z0-9_-]+', 'https://discord.com/api/webhooks/XXXXX/XXXXX'),
    # Email addresses
    (r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', 'user@example.com'),
]

# Patterns to verify are absent after sanitization
VERIFY_PATTERNS = [
    r'jramirez',
    r'ot_[a-zA-Z0-9]{10,}',
    r'-----BEGIN.*KEY-----',
    r'\d{9,}:[A-Za-z0-9_-]{30,}',  # Telegram tokens
]


def sanitize_line(line: str) -> str:
    """Apply all replacements to a single line."""
    for old, new in REPLACEMENTS.items():
        line = line.replace(old, new)
    for pattern, replacement in REGEX_PATTERNS:
        line = re.sub(pattern, replacement, line)
    return line


def sanitize_cast(input_path: str, output_path: str) -> None:
    """Read a .cast file, sanitize all text, write to output."""
    with open(input_path, 'r') as f:
        lines = f.readlines()

    sanitized = []
    for i, line in enumerate(lines):
        sanitized.append(sanitize_line(line))

    with open(output_path, 'w') as f:
        f.writelines(sanitized)

    print(f"Sanitized {len(lines)} lines → {output_path}")


def verify(path: str) -> bool:
    """Check that no sensitive patterns remain in the output."""
    with open(path, 'r') as f:
        content = f.read()

    clean = True
    for pattern in VERIFY_PATTERNS:
        matches = re.findall(pattern, content)
        if matches:
            print(f"  FAIL: pattern '{pattern}' found {len(matches)} times: {matches[:3]}")
            clean = False

    if clean:
        print("  PASS: no sensitive patterns found")
    return clean


def main():
    if len(sys.argv) < 3:
        print("Usage: python3 sanitize-cast.py input.cast output.cast [--verify]")
        sys.exit(1)

    input_path = sys.argv[1]
    output_path = sys.argv[2]
    do_verify = '--verify' in sys.argv

    sanitize_cast(input_path, output_path)

    if do_verify:
        print("\nVerifying output...")
        if not verify(output_path):
            print("\nWARNING: Sensitive patterns remain! Add them to REPLACEMENTS dict.")
            sys.exit(1)


if __name__ == '__main__':
    main()
