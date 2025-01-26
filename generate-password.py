#!/usr/bin/env python3

import sys
import hashlib
import base64

def hash_password(password):
    hasher = hashlib.sha256()
    hasher.update(password.encode('utf-8'))
    return base64.b64encode(hasher.digest()).decode('utf-8')

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: ./generate-password.py PASSWORD")
        sys.exit(1)

    password = sys.argv[1]
    hashed = hash_password(password)
    print(f"Original password: {password}")
    print(f"Hashed password: {hashed}")
    print("\nUse this in your ACCESS_LIST environment variable like:")
    print(f"ACCESS_LIST=username:{hashed}")