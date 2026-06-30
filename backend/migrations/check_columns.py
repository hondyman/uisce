import psycopg2
import os

# Connect to database using env variables
db_url = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"

conn = psycopg2.connect(db_url)
cur = conn.cursor()

# Get columns of notifications table
cur.execute("""
    SELECT column_name, data_type 
    FROM information_schema.columns 
    WHERE table_name = 'notifications'
""")
columns = cur.fetchall()
print("Columns in 'notifications' table:")
for col in columns:
    print(f"  {col[0]} ({col[1]})")

cur.close()
conn.close()
