#!/usr/bin/env python3
import os
import sys
import uuid
import json

if len(sys.argv) < 3:
    print("usage: genconfig <count> server1 [server2 [...]]", file=sys.stderr)
    sys.exit(2)

count = int(sys.argv[1])
servers = sys.argv[2:]


json.dump({
    "BaseURL": "https://friends.ese.ifsr.de/",
    "Port": "localhost:8787",
    "Freelist": {
        "urls": [server + str(uuid.uuid4()) for _ in range(count) for server in servers]
    }
},
sys.stdout,
indent=2)
