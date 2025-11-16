package rabbitmqping

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	pkgpinger "example.com/rbmq-demo/pkg/pinger"
	pkgrbmqrpc "example.com/rbmq-demo/pkg/rpc"
	pkgsimpleping "example.com/rbmq-demo/pkg/simpleping"
)

// RabbitMQPinger is also an implementation of the Pinger interface
type RabbitMQPinger struct {
	From             string
	PingCfg          pkgsimpleping.PingConfiguration
	RBMQRemoteCaller *pkgrbmqrpc.RabbitMQRemoteCaller
}

func (rbmqPinger *RabbitMQPinger) Ping(ctx context.Context) <-chan pkgpinger.PingEvent {
	caller := rbmqPinger.RBMQRemoteCaller

	evChan := make(chan pkgpinger.PingEvent)

	metadata := map[string]string{
		pkgpinger.MetadataKeyFrom: rbmqPinger.From,
	}

	respondWithError := func(err error) {
		log.Println("RabbitMQPinger error:", err)
		ev := pkgpinger.PingEvent{
			Type:     pkgpinger.PingEventTypeError,
			Error:    err,
			Metadata: metadata,
		}
		evChan <- ev
	}

	go func() {
		defer close(evChan)

		if caller == nil {
			respondWithError(fmt.Errorf("the RabbitMQ remote caller is not set within RabbitMQPinger"))
			return
		}

		msgBody, err := json.Marshal(rbmqPinger.PingCfg)
		if err != nil {
			respondWithError(fmt.Errorf("failed to marshal the ping configuration within RabbitMQPinger: %w", err))
			return
		}
		ctx, cancel := context.WithTimeout(ctx, rbmqPinger.PingCfg.Timeout)
		defer cancel()
		for msg := range caller.Call(ctx, msgBody) {
			if msg.Err != nil {
				respondWithError(fmt.Errorf("received an error from underlying RPC call: %w", msg.Err))
				return
			}

			var pingEvent pkgpinger.PingEvent
			err := json.Unmarshal(msg.Message, &pingEvent)
			if err != nil {
				respondWithError(fmt.Errorf("failed to unmarshal the message: %w", err))
				continue
			}

			pingEvent.Metadata = metadata
			evChan <- pingEvent
		}
	}()

	return evChan
}
