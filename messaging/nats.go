package messaging

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func ConnectNATS(natsURL string) (*nats.Conn, error) {
	nc, err := nats.Connect(natsURL, nats.ReconnectWait(10*time.Second), nats.MaxReconnects(5))
	if err != nil {
		return nil, err
	}
	log.Println("Conectado ao NATS em", natsURL)
	return nc, nil
}

func SetupJetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "TELEMETRY",
		Subjects: []string{"telemetry.*"},
	})
	if err != nil {
		if err != nats.ErrStreamNameAlreadyInUse {
			return nil, err
		}
	}
	log.Println("NATS JetStream configurado e pronto.")
	return js, nil
}
