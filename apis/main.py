from fastapi import FastAPI, HTTPException, UploadFile, Request, File
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import boto3
import os
import hashlib
from db import add_user, get_user_state, get_username
import pika
from botocore.exceptions import ClientError
import logging 
from dotenv import load_dotenv

logging.basicConfig(level=logging.INFO)


def s3_test():
    load_dotenv()
    try:
        s3_resource = boto3.resource(
            's3',
            endpoint_url=os.getenv('s3_address'),
            aws_access_key_id=os.getenv('arvan_access_key'),
            aws_secret_access_key=os.getenv('arvan_secret_key')
        )
    except Exception as exc:
        logging.error(exc)
    else:
        try:
            for bucket in s3_resource.buckets.all():
                logging.info(f'bucket_name: {bucket.name}')
        except ClientError as exc:
            logging.error(exc)
        

def get_s3_resource():
    load_dotenv()
    try:
        s3_resource = boto3.resource(
            's3',
            endpoint_url=os.getenv('s3_address'),
            aws_access_key_id=os.getenv('arvan_access_key'),
            aws_secret_access_key=os.getenv('arvan_secret_key')
        )

    except Exception as exc:
        logging.error(exc)
    return s3_resource

def get_s3_client():
    load_dotenv()
    try:
        s3_client = boto3.client(
            's3',
            endpoint_url=os.getenv('s3_address'),
            aws_access_key_id=os.getenv('arvan_access_key'),
            aws_secret_access_key=os.getenv('arvan_secret_key')
        )
        return s3_client
    except Exception as exc:
        logging.error(exc)
        raise

def upload_file_to_s3(s3_client, file: UploadFile, object_name: str):
    try:
        s3_client.upload_fileobj(
            Fileobj=file.file,
            Bucket='image-1bucket',
            Key=object_name,
            ExtraArgs={'ACL': 'private'}
        )
    except ClientError as e:
        logging.error(e)
        raise

app = FastAPI()
@app.get("/test")
async def test():
    return {"Hello": "World"}
    image2: UploadFile

class SubmitResponseModel(BaseModel):
    msg:str

@app.post("/submit-request/",response_model=SubmitResponseModel)
async def submit(email:str, nID:str, last_name:str, request: Request, image1: UploadFile = File(...), image2: UploadFile = File(...)):
    cIP = request.client.host
    username = email.split("@")[0] + last_name

    image1_key = f'{username}_image1.jpg'
    image2_key = f'{username}_image2.jpg'

    hashed_id = hashlib.sha256(nID.encode()).hexdigest()

    state = "pending"
    user = (hashed_id,username,email,last_name,cIP,image1_key,image2_key,state)
    s3_client = get_s3_client()
    if not s3_client:
        return JSONResponse(status_code=500, content={"detail": "Could not initialize S3 client"})
    upload_file_to_s3(s3_client, image1, image1_key)
    upload_file_to_s3(s3_client, image2, image2_key)
    send_to_queue(username)
    add_user(user)
    return {"Response": "Your request has submitted"}



def send_to_queue(username:str):
    load_dotenv()
    params = pika.URLParameters(os.getenv('ampq_url'))

    connection = pika.BlockingConnection(params)

    channel = connection.channel()
    channel.queue_declare(queue='username_queue',durable=True)
    channel.basic_publish(exchange='',routing_key='username_queue',body=username,)
    print(" [x] Sent 'Hello World!'")

    connection.close()


class StatusRequest(BaseModel):
    national_id: str

class StatusResponse(BaseModel):
    username: str
    status: str


@app.get("/status/{national_id}", response_model=StatusResponse)
async def check_status(national_id: str):
    hashed_nid = hashlib.sha256(national_id.encode()).hexdigest()
    status = get_user_state(hashed_nid)
    if status is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="User not found")
    if status == "accepted":
        username = get_username(hashed_nid)
        return JSONResponse(status_code=200,content={"your verified with username: f{username}"})
    return {"status": status}


send_to_queue("test_username2")

# def callback(ch, method, properties, body):
#     print(f" [x] Received {body}")

# def consume_from_queue():
#     load_dotenv()
#     params = pika.URLParameters(os.getenv('ampq_url'))
#     connection = pika.BlockingConnection(params)
#     channel = connection.channel()
    
#     channel.queue_declare(queue='username_queue')
    
#     channel.basic_consume(queue='username_queue', on_message_callback=callback, auto_ack=True)
    
#     print(' [*] Waiting for messages. To exit press CTRL+C')
#     channel.start_consuming()
