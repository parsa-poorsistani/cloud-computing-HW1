FROM python:3.8-slim-buster

WORKDIR /usr/src/app

COPY . .

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install uvicorn

# Make port 80 available to the world outside this container
EXPOSE 8000

# Run app.py when the container launches
CMD ["uvicorn", "main:app", "--host","0.0.0.0", "--port","8000"]