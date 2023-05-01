package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var sess *session.Session

type Member struct {
	Email  string  `json:"email"`
	Weight float64 `json:"weight"`
	Height float64 `json:"height"`
}

func updateMember(member Member) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))
	db := dynamodb.New(sess)

	av, err := dynamodbattribute.MarshalMap(member)
	if err != nil {
		return fmt.Errorf("dynamodb marshal map error:%s", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("MEMBER_TABLE")),
		Item:      av,
	}

	_, err = db.PutItem(input)
	if err != nil {
		return fmt.Errorf("dynamodb put error:%s", err)
	}
	return nil
}

func saveData(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("body:", request.Body)

	var member Member
	if err := json.Unmarshal([]byte(request.Body), &member); err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
		}, nil
	}

	if err := updateMember(member); err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.HTTPMethod {
	case http.MethodPost:
		return saveData(request)
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Access-Control-Allow-Methods": "POST,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type",
			"Content-Type": "application/json",
		},
	}, nil

}

func main() {
	lambda.Start(handler)
}
