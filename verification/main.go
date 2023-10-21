package main

import (
	"bytes"
	"database/sql"
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
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

const (
	Sender  = "poorsistani13@gmail.com"
	CharSet = "UTF-8"
)

func sendEmail(to string, subject string, body string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})
	if err != nil {
		return err
	}

	sesClient := ses.New(sess)

	emailParams := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{aws.String(to)},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data: aws.String(subject),
			},
			Body: &ses.Body{
				Text: &ses.Content{
					Data: aws.String(body),
				},
			},
		},
		Source: aws.String(Sender),
	}

	_, err = sesClient.SendEmail(emailParams)
	return err
}

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
			Score float64 `json:"score"`
		} `json:"result"`
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

func getUserEmail(username string) (string, error) {
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		return "", fmt.Errorf("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", fmt.Errorf("failed to open database connection: %w", err)
	}
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Printf("failed to close database connection: %v", closeErr)
		}
	}()

	var email string
	err = db.QueryRow("SELECT email FROM users WHERE username = $1", username).Scan(&email)
	if err != nil {
		return "", fmt.Errorf("failed to get user email: %w", err)
	}

	return email, nil
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
	faceID1, err := faceDetection(i1)
	if err != nil {
		if faceID1 == "" {
			updateUserState(username, "rejected")
			return
		}
	}

	faceID2, err := faceDetection(i2)
	if err != nil {
		if faceID2 == "" {
			updateUserState(username, "rejected")
			return
		}
	}
	// 3. Compute similarity using face similarity service
	similarityScore, err := faceSimilarity(faceID1, faceID2)
	if err != nil {
		log.Fatal(err)
		return
	}
	var verificationResult string
	if similarityScore < 80 {
		err = updateUserState(username, "rejected")
		if err != nil {
			log.Fatal(err)
		}
		verificationResult = "rejected"
	} else {
		err = updateUserState(username, "accepted")
		if err != nil {
			log.Fatal(err)
		}
		verificationResult = "accepted"
	}

	// 4. If similarity > 80%, send email and update database

	if err != nil {
		log.Printf("Error updating database for username %s: %v", username, err)
		return
	}
	//send email
	userEmail, err := getUserEmail(username)
	if err != nil {
		log.Printf("Error fetching email for username %s: %v", username, err)
		return
	}

	subject := fmt.Sprintf("Verification Result for %s", username)
	body := fmt.Sprintf("Your verification has been %s. Similarity Score: %.2f", verificationResult, similarityScore)
	err = sendEmail(userEmail, subject, body)
	if err != nil {
		log.Printf("Error sending email to user %s: %v", username, err)
	}
}

func updateUserState(username string, newState string) error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return fmt.Errorf("DB_CONNECTION_STRING environment variable is not set")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Printf("failed to close database connection: %v", closeErr)
		}
	}()
	_, err = db.Exec("UPDATE users SET state=$1 WHERE username=$2", newState, username)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	return nil
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
