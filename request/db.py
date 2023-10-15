import psycopg2
import os
from typing import Tuple
from dotenv import load_dotenv


def connect():
    load_dotenv()
    conn = psycopg2.connect(os.getenv('DATABASE_URL'))

    query_sql = 'SELECT VERSION()'

    cur = conn.cursor()
    cur.execute(query_sql)

    version = cur.fetchone()[0]
    print(version)

def connect_db():
    return psycopg2.connect(os.getenv('DATABASE_URL'))

def add_user(user: Tuple[int,str, str, str, str, str, str, str]):
    conn = connect_db()
    cursor = conn.cursor()
    cursor.execute("""
            INSERT INTO users (national_id, username, email, last_name,ip,image_url1,image_url2, state)
            VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
        """, user)
    conn.commit()
    conn.close()

def get_user_state(nID: str) -> str:
    conn = connect_db()
    cursor = conn.cursor()
    cursor.execute("SELECT state FROM users WHERE national_id = %s", (nID,))
    result = cursor.fetchone()
    conn.close()
    return result

def get_username(nID:str) -> str:
    conn = connect_db()
    cursor = conn.cursor()
    cursor.execute("SELECT username FROM users WHERE national_id = %s", (nID,))
    result = cursor.fetchone()
    conn.close()
    return result
