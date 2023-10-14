from fastapi import FastAPI, UploadFile, File, Request
from pydantic import BaseModel
import boto3
import psycopg2
from db import connect

app = FastAPI()

class UserInfo(BaseModel):
    email:str
    last_name:str
    nID:int

class ResponseModel(BaseModel):
    msg:str

class UserModel:
    username: str
    email: str
    ip: str
    state: str
    i1: str
    i2: str
    nID: int
    last_name: str

@app.post("/submit-request/",response_model=ResponseModel)
async def submit(user_info: UserInfo, image1: UploadFile, image2: UploadFile, request: Request):
    cIP = request.client.host
    username = user_info.email.split("@")[0] + user_info.last_name

    


# def main():
#     connect()

# if __name__ == "__main__":
#     main()