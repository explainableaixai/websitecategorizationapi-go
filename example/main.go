package main

import (
	"context"
	"fmt"
	client "github.com/explainableaixai/websitecategorizationapi-go"
	"os"
)

func main() {
	c := client.New(os.Getenv("AQ_API_KEY"))
	result, err := c.Classify(context.Background(), "https://example.com/article")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", result)
}
