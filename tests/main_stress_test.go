package main_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/prom3t3us/turbocookedrabbit/consumer"
	"github.com/prom3t3us/turbocookedrabbit/models"
	"github.com/prom3t3us/turbocookedrabbit/publisher"
	"github.com/prom3t3us/turbocookedrabbit/utils"
	"github.com/stretchr/testify/assert"
)

func TestStressPublishConsumeAckForDuration(t *testing.T) {

	timeDuration := time.Duration(2 * time.Hour)
	timeOut := time.After(timeDuration)
	fmt.Printf("%s: Benchmark Starts\r\n", time.Now())
	fmt.Printf("%s: Est. Benchmark End\r\n", time.Now().Add(timeDuration))

	publisher, err := publisher.NewPublisher(Seasoning, ChannelPool, nil)
	if err != nil {
		assert.NoError(t, err)
		return
	}

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-Ackable"]
	if !ok {
		assert.True(t, ok)
		return
	}
	consumer, conErr := consumer.NewConsumerFromConfig(consumerConfig, ChannelPool)
	if conErr != nil {
		assert.NoError(t, conErr)
		return
	}

	publisher.StartAutoPublish(false)

	go publish(timeOut, publisher)

	if err = consumer.StartConsuming(); err != nil {
		t.Error(err)
	}

	done := make(chan bool, 1)
	go monitor(done, publisher, consumer)

	// Stop RabbitMQ server after entering loop, then start it again, to test reconnectivity.
	processConsumerMessages(timeOut, consumer)

	done <- true

	publisher.StopAutoPublish()

	if err = consumer.StopConsuming(false, true); err != nil {
		t.Error(err)
	}

	ChannelPool.Shutdown()

	fmt.Printf("%s: Benchmark Finished\r\n", time.Now())
}

func publish(timeOut <-chan time.Time, publisher *publisher.Publisher) {

	letter := utils.CreateMockRandomLetter("ConsumerTestQueue")

PublishLoop:
	for {
		select {
		case <-timeOut:
			break PublishLoop
		default:
			newLetter := models.Letter(*letter)
			publisher.QueueLetter(&newLetter)
			//fmt.Printf("%s: Letter Queued - LetterID: %d\r\n", time.Now(), newLetter.LetterID)
			letter.LetterID++
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func processConsumerMessages(timeOut <-chan time.Time, consumer *consumer.Consumer) {

	messagesReceived := 0
	messagesAcked := 0
	messagesFailedToAck := 0

ConsumeLoop:
	for {
		select {
		case <-timeOut:
			break ConsumeLoop
		case message := <-consumer.Messages():
			messagesReceived++
			//fmt.Printf("%s: ConsumedMessage\r\n", time.Now())
			go func(msg *models.Message) {
				err := msg.Acknowledge()
				if err != nil {
					//fmt.Printf("%s: AckMessage Error - %s\r\n", time.Now(), err)
					messagesFailedToAck++
				} else {
					//fmt.Printf("%s: AckMessaged\r\n", time.Now())
					messagesAcked++
				}
			}(message)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	fmt.Printf("Messages Acked: %d\r\n", messagesAcked)
	fmt.Printf("Messages Failed to Ack: %d\r\n", messagesFailedToAck)
	fmt.Printf("Messages Received: %d\r\n", messagesReceived)
}

func monitor(finish <-chan bool, publisher *publisher.Publisher, consumer *consumer.Consumer) {

	messagesPublished := 0
	messagesFailedToPublish := 0
	consumerErrors := 0
	channelPoolErrors := 0
	connectionPoolErrors := 0

NoticeLoop:
	for {
		select {
		case <-finish:
			break NoticeLoop
		case notice := <-publisher.Notifications():
			if notice.Success {
				//fmt.Printf("%s: Published Success - LetterID: %d\r\n", time.Now(), notice.LetterID)
				messagesPublished++
			} else {
				fmt.Printf("%s: Published Failed Error - LetterID: %d\r\n", time.Now(), notice.LetterID)
				messagesFailedToPublish++
			}
		case err := <-ChannelPool.Errors():
			fmt.Printf("%s: ChannelPool Error - %s\r\n", time.Now(), err)
			channelPoolErrors++
		case err := <-ConnectionPool.Errors():
			fmt.Printf("%s: ConnectionPool Error - %s\r\n", time.Now(), err)
			connectionPoolErrors++
		case err := <-consumer.Errors():
			fmt.Printf("%s: Consumer Error - %s\r\n", time.Now(), err)
			consumerErrors++
		}
	}

	fmt.Printf("ChannelPool Errors: %d\r\n", channelPoolErrors)
	fmt.Printf("ConnectionPool Errors: %d\r\n", connectionPoolErrors)
	fmt.Printf("Consumer Errors: %d\r\n", consumerErrors)
	fmt.Printf("Messages Published: %d\r\n", messagesPublished)
	fmt.Printf("Messages Failed to Publish: %d\r\n", messagesFailedToPublish)
}
