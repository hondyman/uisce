package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSQueueBroker polls and publishes via AWS SQS.
type SQSQueueBroker struct{}

func (SQSQueueBroker) client(ctx context.Context, region string) (*sqs.Client, error) {
	opts := []func(*config.LoadOptions) error{}
	if strings.TrimSpace(region) != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return sqs.NewFromConfig(cfg), nil
}

func (b SQSQueueBroker) queueURL(cfg QueueSourceConfig) string {
	if strings.TrimSpace(cfg.TopicOrQueue) != "" {
		return strings.TrimSpace(cfg.TopicOrQueue)
	}
	return envOr(cfg.QueueURLEnv, "AWS_SQS_QUEUE_URL")
}

func (b SQSQueueBroker) sinkURL(cfg QueueSinkConfig) string {
	if strings.TrimSpace(cfg.TopicOrQueue) != "" {
		return strings.TrimSpace(cfg.TopicOrQueue)
	}
	return envOr(cfg.QueueURLEnv, "AWS_SQS_QUEUE_URL")
}

func (b SQSQueueBroker) Poll(ctx context.Context, cfg QueueSourceConfig) ([]map[string]any, error) {
	url := b.queueURL(cfg)
	if url == "" {
		return nil, fmt.Errorf("SQS queue URL not set (topic_or_queue or env AWS_SQS_QUEUE_URL)")
	}
	cli, err := b.client(ctx, cfg.Region)
	if err != nil {
		return nil, err
	}
	max := cfg.MaxMessages
	if max <= 0 {
		max = 1000
	}
	if max > 10 {
		// SQS ReceiveMessage max is 10 per call; loop.
	}
	wait := int32(3)
	if cfg.IdleTimeoutMS > 0 {
		wait = int32(cfg.IdleTimeoutMS / 1000)
		if wait < 1 {
			wait = 1
		}
		if wait > 20 {
			wait = 20
		}
	}

	out := make([]map[string]any, 0, max)
	for len(out) < max {
		n := int32(max - len(out))
		if n > 10 {
			n = 10
		}
		resp, err := cli.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(url),
			MaxNumberOfMessages: n,
			WaitTimeSeconds:     wait,
			VisibilityTimeout:   60,
		})
		if err != nil {
			return out, err
		}
		if len(resp.Messages) == 0 {
			break
		}
		for _, m := range resp.Messages {
			body := ""
			if m.Body != nil {
				body = *m.Body
			}
			row, err := decodeQueueJSON([]byte(body))
			if err != nil {
				row = map[string]any{"_raw": body}
			}
			out = append(out, row)
			if m.ReceiptHandle != nil {
				_, _ = cli.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(url),
					ReceiptHandle: m.ReceiptHandle,
				})
			}
		}
	}
	return out, nil
}

func (b SQSQueueBroker) Publish(ctx context.Context, cfg QueueSinkConfig, rows []map[string]any) (int, error) {
	url := b.sinkURL(cfg)
	if url == "" {
		return 0, fmt.Errorf("SQS queue URL not set (topic_or_queue or env AWS_SQS_QUEUE_URL)")
	}
	cli, err := b.client(ctx, cfg.Region)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		payload, err := json.Marshal(row)
		if err != nil {
			return n, err
		}
		if _, err := cli.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(url),
			MessageBody: aws.String(string(payload)),
		}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
