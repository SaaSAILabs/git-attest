import sqlite3
import os

db_path = os.path.expanduser('~/.gemini/antigravity-ide/conversations/81e15b8e-2630-46e6-a12c-b1c0d1c27c56.db')
conn = sqlite3.connect(db_path)
c = conn.cursor()

c.execute("SELECT step_payload FROM steps WHERE step_type = 14 LIMIT 10")
for row in c.fetchall():
    payload = row[0]
    print(repr(payload))
