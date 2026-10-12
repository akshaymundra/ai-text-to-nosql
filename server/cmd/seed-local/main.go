package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joho/godotenv"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
)

const (
	defaultEndpoint = "http://localhost:8081"
	batchSize       = 25
	waitTimeout     = time.Minute
)

var scalarTypes = map[string]types.ScalarAttributeType{
	"string": types.ScalarAttributeTypeS,
	"number": types.ScalarAttributeTypeN,
}

var localHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

func main() {
	ctx := context.Background()
	_ = godotenv.Load()

	tables, err := dynamoschema.LoadDir("schemas-local")
	if err != nil {
		log.Fatal(err)
	}

	client, err := newClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, t := range tables {
		if err := recreateTable(ctx, client, t); err != nil {
			log.Fatalf("%s: %v", t.Name, err)
		}
		if err := insertRecords(ctx, client, t); err != nil {
			log.Fatalf("%s: %v", t.Name, err)
		}
		log.Printf("%s: created, %d records inserted", t.Name, len(t.ExampleRecords))
	}
}

func newClient(ctx context.Context) (*dynamodb.Client, error) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if err := requireLocal(endpoint); err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-south-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")),
	)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	}), nil
}

func requireLocal(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	if !localHosts[u.Hostname()] {
		return fmt.Errorf("refusing to seed non-local endpoint %q: seeding drops every table", endpoint)
	}
	return nil
}

func recreateTable(ctx context.Context, client *dynamodb.Client, t dynamoschema.Table) error {
	_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(t.Name)})
	var notFound *types.ResourceNotFoundException
	switch {
	case err == nil:
		waiter := dynamodb.NewTableNotExistsWaiter(client)
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(t.Name)}, waitTimeout); err != nil {
			return err
		}
	case !errors.As(err, &notFound):
		return err
	}

	input, err := createTableInput(t)
	if err != nil {
		return err
	}
	if _, err := client.CreateTable(ctx, input); err != nil {
		return err
	}

	waiter := dynamodb.NewTableExistsWaiter(client)
	return waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(t.Name)}, waitTimeout)
}

func createTableInput(t dynamoschema.Table) (*dynamodb.CreateTableInput, error) {
	definitions, err := attributeDefinitions(t)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.CreateTableInput{
		TableName:            aws.String(t.Name),
		KeySchema:            keySchema(t.PartitionKey, t.SortKey),
		AttributeDefinitions: definitions,
		BillingMode:          types.BillingModePayPerRequest,
	}
	for _, idx := range t.Indexes {
		input.GlobalSecondaryIndexes = append(input.GlobalSecondaryIndexes, types.GlobalSecondaryIndex{
			IndexName:  aws.String(idx.Name),
			KeySchema:  keySchema(idx.PartitionKey, idx.SortKey),
			Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
		})
	}
	return input, nil
}

func keySchema(partitionKey, sortKey string) []types.KeySchemaElement {
	keys := []types.KeySchemaElement{{AttributeName: aws.String(partitionKey), KeyType: types.KeyTypeHash}}
	if sortKey != "" {
		keys = append(keys, types.KeySchemaElement{AttributeName: aws.String(sortKey), KeyType: types.KeyTypeRange})
	}
	return keys
}

// DynamoDB rejects definitions for attributes that are not part of any key, so only key attributes are declared.
func attributeDefinitions(t dynamoschema.Table) ([]types.AttributeDefinition, error) {
	keyNames := []string{t.PartitionKey, t.SortKey}
	for _, idx := range t.Indexes {
		keyNames = append(keyNames, idx.PartitionKey, idx.SortKey)
	}

	seen := map[string]bool{"": true}
	var definitions []types.AttributeDefinition
	for _, name := range keyNames {
		if seen[name] {
			continue
		}
		seen[name] = true

		attr, ok := t.Attribute(name)
		if !ok {
			return nil, fmt.Errorf("key attribute %q is not defined in attributes", name)
		}
		scalarType, ok := scalarTypes[attr.Type]
		if !ok {
			return nil, fmt.Errorf("key attribute %q has unsupported type %q", name, attr.Type)
		}
		definitions = append(definitions, types.AttributeDefinition{AttributeName: aws.String(name), AttributeType: scalarType})
	}
	return definitions, nil
}

func insertRecords(ctx context.Context, client *dynamodb.Client, t dynamoschema.Table) error {
	for start := 0; start < len(t.ExampleRecords); start += batchSize {
		end := min(start+batchSize, len(t.ExampleRecords))

		requests := make([]types.WriteRequest, 0, end-start)
		for _, record := range t.ExampleRecords[start:end] {
			item, err := attributevalue.MarshalMap(record)
			if err != nil {
				return err
			}
			requests = append(requests, types.WriteRequest{PutRequest: &types.PutRequest{Item: item}})
		}

		pending := map[string][]types.WriteRequest{t.Name: requests}
		for len(pending) > 0 {
			out, err := client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{RequestItems: pending})
			if err != nil {
				return err
			}
			pending = out.UnprocessedItems
		}
	}
	return nil
}
