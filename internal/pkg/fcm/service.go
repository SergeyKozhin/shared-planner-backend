package fcm

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	client *messaging.Client
}

func NewService(ctx context.Context) (*Service, error) {
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtaining messaging client: %w", err)
	}

	return &Service{client: client}, nil
}

type Message struct {
	Token string
	Data  map[string]string
}

func (s *Service) SendMessage(ctx context.Context, m *Message) error {
	message := &messaging.Message{
		Data:  m.Data,
		Token: m.Token,
	}

	_, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

const batchSize = 500

func (s *Service) SendMessageBatch(ctx context.Context, ms []*Message) error {
	messages := make([]*messaging.Message, len(ms))
	for i, m := range ms {
		messages[i] = &messaging.Message{
			Data:  m.Data,
			Token: m.Token,
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < len(messages); i += batchSize {
		from := i
		to := i + batchSize
		if to > len(messages) {
			to = len(messages)
		}

		g.Go(func() error {
			_, err := s.client.SendAll(ctx, messages[from:to])
			if err != nil {
				return fmt.Errorf("send message: %w", err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
