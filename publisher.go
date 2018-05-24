package loglet

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"

	"github.com/uswitch/loglet/cmd/loglet/options"
)

type Publisher interface {
	Ret() <-chan error
	Published() <-chan string
}

type kinesisPublisher struct {
	client       *kinesis.Kinesis
	stream       string
	partitionKey string
	ret          chan error
	published    chan string
}

func NewKinesisPublisher(loglet *options.Loglet, msgs <-chan *EncodedMessage, done <-chan struct{}) (Publisher, error) {

	// create client
	svc := kinesis.New(session.Must(session.NewSession()), aws.NewConfig().WithRegion(loglet.KinesisRegion))

	publisher := &kinesisPublisher{
		client:       svc,
		stream:       loglet.KinesisStream,
		partitionKey: loglet.KinesisPartitionKey,
		ret:          make(chan error),
		published:    make(chan string),
	}

	go publisher.loop(msgs, done)

	return publisher, nil
}

func (p *kinesisPublisher) Ret() <-chan error {
	return p.ret
}

func (p *kinesisPublisher) Published() <-chan string {
	return p.published
}

func (p *kinesisPublisher) loop(msgs <-chan *EncodedMessage, done <-chan struct{}) {
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
			_, err := p.client.PutRecord(&kinesis.PutRecordInput{
				StreamName:   &p.stream,
				PartitionKey: &p.partitionKey,
				Data:         m.Message,
			})
			if err != nil {
				p.ret <- fmt.Errorf("kinesis: unable to produce message: %v", err)
				return
			}
		}

		select {
		case <-done:
			return
		case p.published <- m.Cursor:
		}
	}

}
