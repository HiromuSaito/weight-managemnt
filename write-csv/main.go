package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jszwec/csvutil"
)

var sess *session.Session
var region = "ap-northeast-1"

type Member struct {
	Email  string  `csv:"email" json:"email"`
	Name   string  `csv:"name" json:"name"`
	Weight float64 `json:"weight"`
	Height float64 `json:"height"`
}

func getMembers() ([]Member, error) {
	db := dynamodb.New(sess)
	scanOut, err := db.Scan(&dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("MEMBER_TABLE")),
	})

	if err != nil {
		log.Printf("dynamo scan error:%s", err)
		return nil, err
	}

	var members []Member = []Member{}
	for _, scanedMember := range scanOut.Items {
		var memberTmp Member
		_ = dynamodbattribute.UnmarshalMap(scanedMember, &memberTmp)
		members = append(members, memberTmp)
	}

	return members, nil
}
func createCsv(members []Member) (string, error) {
	b, err := csvutil.Marshal(members)
	if err != nil {
		log.Println("csv unmarshal error:", err)
		return "", err
	}
	fmt.Println(string(b))

	fileName := time.Now().Format("20060102150405.csv")
	filePath := "/tmp/" + fileName
	err = os.WriteFile(filePath, b, 0666)
	if err != nil {
		log.Println("csv write error:", err)
		return "", nil
	}
	return filePath, nil
}
func uploadCsv(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("csv open error", err)
		return err
	}
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("CALCULATE_BUCKET")),
		Key:    aws.String(strings.Replace(filePath, "/tmp", "", -1)),
		Body:   file,
	})
	if err != nil {
		log.Println("csv upload error", err)
		return err
	}
	return nil
}

func handler(event events.CloudWatchEvent) {
	log.Println("---------------start---------------------")
	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	members, err := getMembers()
	if err != nil {
		return
	}

	filePath, err := createCsv(members)
	if err != nil {
		return
	}
	err = uploadCsv(filePath)
	if err != nil {
		return
	}
	log.Println("---------------finish---------------------")
}

func main() {
	lambda.Start(handler)
}
