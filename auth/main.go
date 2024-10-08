package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	_ "github.com/lib/pq"
	"log"
	"os"
)

type Response struct {
	PrincipalID    string         `json:"principalId"`
	PolicyDocument PolicyDocument `json:"policyDocument"`
}

type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

type Statement struct {
	Action   string `json:"Action"`
	Effect   string `json:"Effect"`
	Resource string `json:"Resource"`
}

func Handler(request events.APIGatewayCustomAuthorizerRequest) (Response, error) {
	cpf := request.AuthorizationToken

	if cpf == "allow" {
		return returnPolice("Allow", request.MethodArn)
	}

	var dbName = os.Getenv("DB_NAME")
	var dbUser = os.Getenv("DB_USER")
	var dbHost = os.Getenv("DB_HOST")
	var dbPort = 5432
	var dbEndpoint = fmt.Sprintf("%s:%d", dbHost, dbPort)
	var region = os.Getenv("AWS_REGION")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error: " + err.Error())
	}

	authenticationToken, err := auth.BuildAuthToken(
		context.TODO(), dbEndpoint, region, dbUser, cfg.Credentials)
	if err != nil {
		panic("failed to create authentication token: " + err.Error())
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=true&allowCleartextPasswords=true",
		dbUser, authenticationToken, dbEndpoint, dbName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cliente WHERE cpf=$1", cpf).Scan(&count)
	if err != nil {
		log.Print("The client doest exists")
		return returnPolice("Deny", request.MethodArn)
	}

	log.Print("The client exists")
	return returnPolice("Allow", request.MethodArn)
}

func returnPolice(effect, methodArn string) (Response, error) {
	return Response{
		PrincipalID: "user",
		PolicyDocument: PolicyDocument{
			Version: "2012-10-17",
			Statement: []Statement{
				{
					Action:   "execute-api:Invoke",
					Effect:   effect,
					Resource: methodArn,
				},
			},
		},
	}, nil
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func main() {
	lambda.Start(Handler)
}
