package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	kafkaHost := os.Getenv("GOO_KAFKA_HOST")
	logrus.Info("Using host: %s", kafkaHost)
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": kafkaHost})
	if err != nil {
		panic(err)
	}

	fmt.Println(p)

	stopRetrieverChannel := make(chan bool)
	defer close(stopRetrieverChannel)

	periodicMessenger := PeriodicMessenger{
		StopChannel:   stopRetrieverChannel,
		KafkaProducer: p,
	}

	periodicMessenger.Run()

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)

	select {
	case <-sigTerm:
		{
			stopRetrieverChannel <- true
			logrus.Info("Exiting per SIGTERM")
		}
	}
}

type PeriodicMessenger struct {
	StopChannel   <-chan bool
	KafkaProducer *kafka.Producer
}

func (periodicMessenger *PeriodicMessenger) Run() {
	periodic := time.NewTicker(time.Second * 7)

	go func() {
		for {
			select {
			case <-periodicMessenger.StopChannel:
				{
					logrus.Info("Shutting down Slack Retriever")
					return
				}
			case t := <-periodic.C:
				{
					logrus.Info("Tick at ", t)
					periodicMessenger.worker()
				}
			}
		}
	}()
}

func (periodicMessenger *PeriodicMessenger) worker() {
	topic := "test"
	produce := func(message string) {
		periodicMessenger.KafkaProducer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(message),
		}, nil)
	}

	produceError := func(err error) {
		produce(fmt.Sprintf("ERROR: %#v", err))
	}

	url := "http://lexemes.stochastica.xyz/api/words?howMany=4&minWordLength=6"

	if req, err := http.NewRequest("GET", url, nil); err != nil {
		go produceError(err)
	} else {
		client := &http.Client{}
		if resp, err := client.Do(req); err != nil {
			go produceError(err)
		} else {
			defer resp.Body.Close()
			if body, err := ioutil.ReadAll(resp.Body); err != nil {
				go produceError(err)
			} else {
				fmt.Println(string(body))
				var data Data
				if err = json.Unmarshal(body, &data); err != nil {
					go produceError(err)
				} else {
					go produce(strings.Join(data.Data.Words, "-"))
				}
			}
		}
	}
}

type Data struct {
	Data Words
}

type Words struct {
	Words []string
}
