package main

import (
	"context"
	"errors"

	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

const TableName = "Todos"

var db dynamodb.Client

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	db = *dynamodb.NewFromConfig(sdkConfig)
}

type Todo struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Status      bool   `json:"status" dynamodbav:"status"`
}

func getItem(ctx context.Context, id string) (*Todo, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
	}

	log.Printf("Calling Dynamodb with input: %v", input)
	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Printf("Executed GetItem DynamoDb successfully. Result: %#v", result)

	if result.Item == nil {
		return nil, nil
	}

	todo := new(Todo)
	err = attributevalue.UnmarshalMap(result.Item, todo)
	if err != nil {
		return nil, err
	}

	return todo, nil
}

func listItems(ctx context.Context) ([]Todo, error) {
	todos := make([]Todo, 0)
	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			return nil, err
		}

		var fetchedTodos []Todo
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedTodos)
		if err != nil {
			return nil, err
		}

		todos = append(todos, fetchedTodos...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}
	}

	return todos, nil
}

func insertItem(ctx context.Context, createTodo CreateTodo) (*Todo, error) {
	todo := Todo{
		Name:        createTodo.Name,
		Description: createTodo.Description,
		Status:      false,
		Id:          uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(todo)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &todo)
	if err != nil {
		return nil, err
	}

	return &todo, nil
}

func deleteItem(ctx context.Context, id string) (*Todo, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	res, err := db.DeleteItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	todo := new(Todo)
	err = attributevalue.UnmarshalMap(res.Attributes, todo)
	if err != nil {
		return nil, err
	}

	return todo, nil
}

func updateItem(ctx context.Context, id string, updateTodo UpdateTodo) (*Todo, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	expr, err := expression.NewBuilder().WithUpdate(
		expression.Set(
			expression.Name("name"),
			expression.Value(updateTodo.Name),
		).Set(
			expression.Name("description"),
			expression.Value(updateTodo.Description),
		).Set(
			expression.Name("status"),
			expression.Value(updateTodo.Status),
		),
	).WithCondition(
		expression.Equal(
			expression.Name("id"),
			expression.Value(id),
		),
	).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		TableName:                 aws.String(TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValue(*aws.String("ALL_NEW")),
	}

	res, err := db.UpdateItem(ctx, input)
	if err != nil {
		var smErr *smithy.OperationError
		if errors.As(err, &smErr) {
			var condCheckFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condCheckFailed) {
				return nil, nil
			}
		}

		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	todo := new(Todo)
	err = attributevalue.UnmarshalMap(res.Attributes, todo)
	if err != nil {
		return nil, err
	}

	return todo, nil
}
