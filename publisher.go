package loglet

import (
	"fmt"
	"time"

	kafka "github.com/Shopify/sarama"

	"github.com/uswitch/loglet/cmd/loglet/options"
)

type Publisher interface {
	Ret() <-chan error
	Published() <-chan string
}

type kafkaPublisher struct {
	producer  kafka.SyncProducer
	topic     string
	ret       chan error
	published chan string
}

func NewKafkaPublisher(loglet *options.Loglet, msgs <-chan *EncodedMessage, done <-chan struct{}) (Publisher, error) {

	producer, err := createProducer(loglet)
	if err != nil {
		return nil, fmt.Errorf("kafka: unable to create producer: %v", err)
	}

	publisher := &kafkaPublisher{
		ret:       make(chan error),
		published: make(chan string),
		producer:  producer,
		topic:     loglet.KafkaTopic,
	}

	go publisher.loop(msgs, done)

	return publisher, nil
}

func createProducer(loglet *options.Loglet) (kafka.SyncProducer, error) {
	if loglet.FakeKafka {
		return nil, nil
	}

	config := kafka.NewConfig()
	config.ClientID = "loglet"
	config.Producer.RequiredAcks = kafka.WaitForLocal
	//config.Producer.Flush.Messages = kafka.
	config.Producer.Retry.Backoff = 1 * time.Second

	return kafka.NewSyncProducer(loglet.KafkaBrokers, config)
}

func (p *kafkaPublisher) Ret() <-chan error {
	return p.ret
}

func (p *kafkaPublisher) Published() <-chan string {
	return p.published
}

func (p *kafkaPublisher) loop(msgs <-chan *EncodedMessage, done <-chan struct{}) {
	defer close(p.ret)
	defer close(p.published)

	var (
		ok bool
		m  *EncodedMessage
	)

	for {
		select {
		case <-done:
			return
		case m, ok = <-msgs:
			if !ok {
				return
			}
			if p.producer != nil {
				_, _, err := p.producer.SendMessage(&kafka.ProducerMessage{
					Topic: p.topic,
					Value: kafka.ByteEncoder(m.Message),
				})
				if err != nil {
					p.ret <- fmt.Errorf("kafka: unable to produce message: %v", err)
					return
				}
			}
		}

		select {
		case <-done:
			return
		case p.published <- m.Cursor:
		}
	}

}
