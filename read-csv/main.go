package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/jszwec/csvutil"
)

var sess *session.Session
var region = "ap-northeast-1"

type Member struct {
	Email      string `csv:"email" json:"email"`
	Name       string `csv:"name" json:"name"`
	IsMailSend bool   `json:"ismailsend"`
}

func getCsvFromS3(bucketName, objectKey string) (*s3.GetObjectOutput, error) {
	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return nil, fmt.Errorf("%s:%s", s3.ErrCodeNoSuchKey, aerr.Error())
			case s3.ErrCodeInvalidObjectState:
				return nil, fmt.Errorf("%s:%s", s3.ErrCodeInvalidObjectState, aerr.Error())
			default:
				return nil, fmt.Errorf("error:%s", aerr.Error())
			}
		} else {
			return nil, aerr
		}
	}
	return result, nil
}

func s3ObjectToMembers(s3Object *s3.GetObjectOutput) ([]Member, error) {
	defer s3Object.Body.Close()
	var members []Member

	data, err := ioutil.ReadAll(s3Object.Body)
	if err != nil {
		return members, err
	}
	if err = csvutil.Unmarshal(data, &members); err != nil {
		return members, err
	}
	return members, nil
}

func bulkUpdateMember(members []Member) error {
	db := dynamodb.New(sess)
	for _, member := range members {

		av, err := dynamodbattribute.MarshalMap(member)
		if err != nil {
			return fmt.Errorf("dynamodb marshal map error:%s", err)
		}

		input := &dynamodb.PutItemInput{
			TableName: aws.String("members"),
			Item:      av,
		}

		_, err = db.PutItem(input)
		if err != nil {
			return fmt.Errorf("dynamodb put error:%s", err)
		}
	}
	return nil
}

func sendSqsMessage(members []Member) error {
	svc := sqs.New(sess)
	for _, members := range members {
		params := &sqs.SendMessageInput{
			MessageBody:  aws.String(members.Email),
			QueueUrl:     aws.String(os.Getenv("QUEUE_URL")),
			DelaySeconds: aws.Int64(1),
		}
		_, err := svc.SendMessage(params)
		if err != nil {
			return err
		}
	}

	return nil
}

func handler(event events.S3Event) {
	log.Println("---------------start---------------------")

	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	for _, record := range event.Records {
		bucketName := record.S3.Bucket.Name
		objectKey := record.S3.Object.Key

		//s3からcsvを取得
		s3Object, err := getCsvFromS3(bucketName, objectKey)
		if err != nil {
			log.Printf("getCsv From s3 error:%s", err)
			return
		}

		//csvの内容を構造体にマッピング
		members, err := s3ObjectToMembers(s3Object)
		if err != nil {
			log.Printf("csv unmarshal error:%s", err)
		}

		//dynamodbへの登録
		err = bulkUpdateMember(members)
		if err != nil {
			log.Printf("dynamodb update error:%s", err)
		}

		//sqsへの送信
		err = sendSqsMessage(members)
		if err != nil {
			log.Printf("send sqs message error:%s", err)
		}
	}

	log.Println("---------------finish---------------------")
}

func main() {
	lambda.Start(handler)
}
