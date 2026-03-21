package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSClient struct {
	client   *sns.Client
	topicARN string
}

func NewSNSClient(topicARN string) (*SNSClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return &SNSClient{
		client:   sns.NewFromConfig(cfg),
		topicARN: topicARN,
	}, nil
}

func (s *SNSClient) Publish(order *Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return err
	}
	_, err = s.client.Publish(context.Background(), &sns.PublishInput{
		TopicArn: aws.String(s.topicARN),
		Message:  aws.String(string(body)),
	})
	return err
}
