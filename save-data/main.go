package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
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

	update := expression.UpdateBuilder{}.Set(expression.Name("weight"), expression.Value(member.Weight))
	update.Set(expression.Name("height"), expression.Value(member.Height))

	condition := expression.Name("email").Equal(expression.Value(member.Email))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(condition).Build()
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(os.Getenv("MEMBER_TABLE")),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(member.Email),
			},
		},
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              aws.String(dynamodb.ReturnValueAllNew),
	}
	_, err = db.UpdateItem(input)
	return err
}

func saveData(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type",
			"Content-Type":                 "application/json",
		},
	}, nil

}

func main() {
	lambda.Start(handler)
}
