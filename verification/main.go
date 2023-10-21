package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

func connectToRabbitMQ() (*amqp.Connection, error) {
	conn, err := amqp.Dial(os.Getenv("ampq_url"))
	if err != nil {
		log.Fatal(err)
	}
	return conn, err
}

func faceDetection(imageBase64 string) (string, error) {
	apiURL := "https://api.imagga.com/v2/faces/detections"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString("image_base64="+imageBase64))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(os.Getenv("imagga_key"), os.Getenv("imagga_secret"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to detect face: %v, %s", resp.Status, body)
	}

	var result struct {
		Result struct {
			Faces []struct {
				FaceID string `json:"face_id"`
				// Confidence int16 `json:"confidence"`
			} `json:"faces"`
		} `json:"result"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if len(result.Result.Faces) == 0 {
		return "", fmt.Errorf("no faces detected")
	}

	return result.Result.Faces[0].FaceID, nil
}

func faceSimilarity(faceID1, faceID2 string) (float64, error) {
	APIUrl := fmt.Sprintf("https://api.imagga.com/v2/faces/similarity?face_id=%s&second_face_id=%s", faceID1, faceID2)
	req, err := http.NewRequest("GET", APIUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth(os.Getenv("imagga_key"), os.Getenv("imagga_secret"))
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("face similarity failed:  %v, %s", resp.StatusCode, body)
	}
	var result struct {
		Result struct {
			Score float64 `json:score`
		} `json:result`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Result.Score, nil
}
func handleMessages(conn *amqp.Connection) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"username_queue", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatal(err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			// Process the message
			processMessage(string(d.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func getS3ImagesBase64(username string) (string, string, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("arvan_access_key"), os.Getenv("arvan_secret_key"), ""),
		Region:      aws.String("default"),
		Endpoint:    aws.String(os.Getenv("s3_address")),
	})
	if err != nil {
		log.Fatal(err)
	}
	s3Client := s3.New(sess)

	image1key := fmt.Sprintf("%s_image1.jpg", username)
	image2key := fmt.Sprintf("%s_image2.jpg", username)

	image1, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("image-1bucket"),
		Key:    aws.String(image1key),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer image1.Body.Close()
	image2, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("image-1bucket"),
		Key:    aws.String(image2key),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer image2.Body.Close()

	image1Bytes, err := io.ReadAll(image1.Body)
	if err != nil {
		return "", "", err
	}

	image2Bytes, err := io.ReadAll(image2.Body)
	if err != nil {
		return "", "", err
	}

	return string(image1Bytes), string(image2Bytes), nil
}

func processMessage(username string) {
	// 1. Retrieve images from S3
	i1, i2, err := getS3ImagesBase64(username)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Verify faces using face detection service
	
	// 3. Compute similarity using face similarity service
	// 4. If similarity > 80%, send email and update database
}

func updateUserState(username string) {

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	conn, err := connectToRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	handleMessages(conn)
}
