from fastapi import FastAPI, HTTPException, UploadFile, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import boto3
import os
from db import add_user, get_user_state, get_username
import pika
from botocore.exceptions import ClientError
import logging 
from dotenv import load_dotenv

logging.basicConfig(level=logging.INFO)

def s3_test():
    load_dotenv()
    try:
        s3_client = boto3.client(
            's3',
            endpoint_url=os.getenv('s3_address'),
            aws_access_key_id=os.getenv('arvan_access_key'),
            aws_secret_access_key=os.getenv('arvan_secret_key')
        )
    except Exception as exc:
        logging.info(exc)
    else:
        try:
            # for bucket in s3_client.buckets.all():
            #     print(f'bucket_name: {bucket.name}')
            #     logging.info(f'bucket_name: {bucket.name}')
            response = s3_client.head_bucket(Bucket="image-1bucket")
        except ClientError as err:
            # logging.info(err)
            status = err.response["ResponseMetadata"]["HTTPStatusCode"]
            errcode = err.response["Error"]["Code"]

            if status == 404:
                logging.warning("Missing object, %s", errcode)
                return
            elif status == 403:
                logging.error("Access denied, %s", errcode)
                return
            else:
                logging.exception("Error in request, %s", errcode)
                return
        else:
            print(response)
    # return s3_client

def get_s3_resource():
    try:
        s3_resource = boto3.resource(
            's3',
            endpoint_url='endpoint_url',
            aws_access_key_id='access_key',
            aws_secret_access_key='secret_key'
        )

    except Exception as exc:
        logging.error(exc)
    return s3_resource

def get_s3_client():
    try:
        s3_client = boto3.client(
            's3',
            endpoint_url='endpoint_url',
            aws_access_key_id='access_key',
            aws_secret_access_key='secret_key'
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

class UserInfo(BaseModel):
    email:str
    last_name:str
    nID:int

class SubmitResponseModel(BaseModel):
    msg:str


@app.post("/submit-request/",response_model=SubmitResponseModel)
async def submit(user_info: UserInfo, image1: UploadFile, image2: UploadFile, request: Request):
    cIP = request.client.host
    username = user_info.email.split("@")[0] + user_info.last_name

    image1_key = f'{username}_image1.jpg'
    image2_key = f'{username}_image2.jpg'

    state = "pending"
    user = (user_info.nID,username,user_info.email,user_info.last_name,cIP,image1_key,image2_key,state)
    # bucket = get_s3_resource()
    s3_client = get_s3_client()
    if not s3_client:
        return JSONResponse(status_code=500, content={"detail": "Could not initialize S3 client"})
    upload_file_to_s3(s3_client, image1, image1_key)
    upload_file_to_s3(s3_client, image2, image2_key)
    # try:
    #     file_path = f'images/{image1_key}.jpg'
    #     object_name = image1_key

    #     with open(file_path, "rb") as file:
    #         bucket.put_object(
    #             ACL='private',
    #             Body=file,
    #             Key=object_name
    #         )
    # except ClientError as e:
    #     logging.error(e)
    # try:
    #     file_path = f'images/{image2_key}.jpg'
    #     object_name = image2_key

    #     with open(file_path, "rb") as file:
    #         bucket.put_object(
    #             ACL='private',
    #             Body=file,
    #             Key=object_name
    #         )
    # except ClientError as e:
    #     logging.error(e)
    send_to_queue(username)
    add_user(user)
    return JSONResponse(status_code=200,content={"your request has submited"})


def send_to_queue(username:str):
    load_dotenv()

    params = pika.URLParameters(os.getenv('ampq_url'))

    connection = pika.BlockingConnection(params)

    channel = connection.channel()
    channel.queue_declare(queue='username_queue')
    channel.basic_publish(exchange='',routing_key='username_queue',body=username)

    connection.close()


class StatusRequest(BaseModel):
    national_id: str

class StatusResponse(BaseModel):
    username: str
    status: str


@app.post("/status", response_model=StatusResponse)
async def check_status(request: StatusRequest):
    status = get_user_state(request.national_id)
    if status is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="User not found")
    if status == "accepted":
        username = get_username(request.national_id)
        return JSONResponse(status_code=200,content={"your verified with username: f{username}"})
    return {"status": status}

def main():
    load_dotenv()
    # try:
    #     s3_client = boto3.client(
    #         's3',
    #         endpoint_url=os.getenv('s3_address'),
    #         aws_access_key_id=os.getenv('arvan_access_key'),
    #         aws_secret_access_key=os.getenv('arvan_secret_key')
    #     )
    # except Exception as exc:
    #     logging.error(exc)
    # else:
    #     try:
    #         response = s3_client.head_bucket(Bucket="image-1bucket")
    #     except ClientError as err:
    #         status = err.response["ResponseMetadata"]["HTTPStatusCode"]
    #         errcode = err.response["Error"]["Code"]

    #         if status == 404:
    #             logging.warning("Missing object, %s", errcode)
    #         elif status == 403:
    #             logging.error("Access denied, %s", errcode)
    #         else:
    #             logging.exception("Error in request, %s", errcode)
    #     else:
    #         print(response)

if __name__ == "__main__":
    main()