import sqlite3
import os
import datetime

db_path = os.path.expanduser("~/.gemini/antigravity-ide/conversations/81e15b8e-2630-46e6-a12c-b1c0d1c27c56.db")

try:
    # Connect in read-only mode to avoid locking issues
    conn = sqlite3.connect(f"file:{db_path}?mode=ro", uri=True)
    cursor = conn.cursor()

    # Query step_type 14 (User Prompts)
    cursor.execute("SELECT idx, step_payload FROM steps WHERE step_type = 14 AND step_payload IS NOT NULL")
    rows = cursor.fetchall()

    print(f"Found {len(rows)} steps matching step_type = 14\n")

    for idx, payload in rows:
        print(f"Index: {idx}")
        print(f"Payload: {payload}")
        print("-" * 50)
        # Safely peek at the first few bytes in hex to see the protobuf wire structure
        print(f"Hex Preview: {payload.hex()}")

except sqlite3.OperationalError as e:
    print(f"Database error: {e}. Ensure the path to the .db file is correct.")
finally:
    if 'conn' in locals():
        conn.close()