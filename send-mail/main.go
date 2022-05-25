package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
)

var sess *session.Session

func issuePreSignedUrl() (string, error) {
	svc := s3.New(sess)

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("HOSTING_BUCKET")),
		Key:    aws.String("index.html"),
	})

	return req.Presign(time.Hour * 2) // 有効期限を指定して署名付きURLを取得
}

func sendMail(toAddress string, url string) error {
	svc := ses.New(sess)
	fmt.Println("toAddress:", toAddress)
	fmt.Println("FromAddress:", os.Getenv("ADMIN_MAIL_ADDRESS"))

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(toAddress),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(url),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String("体重測定のお願い"),
			},
		},
		Source: aws.String(os.Getenv("ADMIN_MAIL_ADDRESS")),
		// Source: aws.String("hiromu.s08.t27@gmail.com"),
	}
	_, err := svc.SendEmail(input)
	return err
}

func updateSendFlag(email string) error {
	log.Println("updateSendFlag")
	db := dynamodb.New(sess)

	update := expression.UpdateBuilder{}.Set(expression.Name("ismailsend"), expression.Value(true))
	condition := expression.Name("email").Equal(expression.Value(email))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("members"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              aws.String(dynamodb.ReturnValueAllNew),
	}
	// Execute.
	_, err = db.UpdateItem(input)
	return err
}

func handler(event events.SQSEvent) {
	log.Println("SendMailFunction start --------------------")

	sess = session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	for _, record := range event.Records {
		mailAddress := record.Body

		// preSignedURLの発行
		url, err := issuePreSignedUrl()
		if err != nil {
			log.Printf("issue preSigned url error:%s", err)
			continue
		}

		//メール送信
		if err := sendMail(mailAddress, url); err != nil {
			log.Printf("send mail error:%s", err)
			continue
		}

		//DB更新
		if err := updateSendFlag(mailAddress); err != nil {
			log.Printf("update sendFlag error:%s", err)
			continue
		}
	}

	log.Println("SendMailFunction finish --------------------")
}

func main() {
	lambda.Start(handler)
}
