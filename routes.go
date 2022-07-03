package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
)

type UpdateTodo struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Status      bool   `json:"status" validate:"required"`
}

type CreateTodo struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
}

var validate *validator.Validate = validator.New()

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received req %#v", req)

	switch req.HTTPMethod {
	case "GET":
		return processGet(ctx, req)
	case "POST":
		return processPost(ctx, req)
	case "DELETE":
		return processDelete(ctx, req)
	case "PUT":
		return processPut(ctx, req)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func processGet(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return processGetTodos(ctx)
	} else {
		return processGetTodo(ctx, id)
	}
}

func processGetTodo(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received GET todo request with id = %s", id)

	todo, err := getItem(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if todo == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(todo)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched todo item %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processGetTodos(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET todos request")

	todos, err := listItems(ctx)
	if err != nil {
		return serverError(err)
	}

	json, err := json.Marshal(todos)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched todos: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var createTodo CreateTodo
	err := json.Unmarshal([]byte(req.Body), &createTodo)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&createTodo)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received POST request with item: %+v", createTodo)

	res, err := insertItem(ctx, createTodo)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Inserted new todo: %+v", res)

	json, err := json.Marshal(res)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/todo/%s", res.Id),
		},
	}, nil
}

func processDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received DELETE request with id = %s", id)

	todo, err := deleteItem(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if todo == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(todo)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully deleted todo item %+v", todo)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}

	var updateTodo UpdateTodo
	err := json.Unmarshal([]byte(req.Body), &updateTodo)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&updateTodo)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received PUT request with item: %+v", updateTodo)

	res, err := updateItem(ctx, id, updateTodo)
	if err != nil {
		return serverError(err)
	}

	if res == nil {
		return clientError(http.StatusNotFound)
	}

	log.Printf("Updated todo: %+v", res)

	json, err := json.Marshal(res)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/todo/%s", res.Id),
		},
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}
