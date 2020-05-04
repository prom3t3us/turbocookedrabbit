package consumer_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"

	"github.com/prom3t3us/turbocookedrabbit/consumer"
	"github.com/prom3t3us/turbocookedrabbit/models"
	"github.com/prom3t3us/turbocookedrabbit/pools"
	"github.com/prom3t3us/turbocookedrabbit/publisher"
	"github.com/prom3t3us/turbocookedrabbit/utils"
)

var Seasoning *models.RabbitSeasoning
var ConnectionPool *pools.ConnectionPool
var ChannelPool *pools.ChannelPool

func TestMain(m *testing.M) {

	var err error
	Seasoning, err = utils.ConvertJSONFileToConfig("testconsumerseasoning.json") // Load Configuration On Startup
	if err != nil {
		return
	}
	ConnectionPool, err = pools.NewConnectionPool(Seasoning.PoolConfig, true)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	ChannelPool, err = pools.NewChannelPool(Seasoning.PoolConfig, ConnectionPool, true)
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	os.Exit(m.Run())
}

func TestCreateConsumer(t *testing.T) {
	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	con1, err1 := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err1)
	assert.NotNil(t, con1)

	con2, err2 := consumer.NewConsumer(
		Seasoning,
		channelPool,
		"ConsumerTestQueue",
		"MyConsumerName",
		false,
		false,
		false,
		nil,
		0,
		10,
		2,
		5,
		5,
	)
	assert.NoError(t, err2)
	assert.NotNil(t, con2)
}

func TestCreateConsumerAndGet(t *testing.T) {
	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, err := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err)
	assert.NotNil(t, consumer)

	_, err = consumer.Get("ConsumerTestQueue", true)
	assert.NoError(t, err)

	_, err = consumer.Get("ConsumerTestQueue", false)
	assert.NoError(t, err)

	_, err = consumer.Get("ConsumerTestQueue2", false)
	assert.Error(t, err)
}

func TestCreateConsumerAndGetBatch(t *testing.T) {
	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, err := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err)
	assert.NotNil(t, consumer)

	_, err = consumer.GetBatch("ConsumerTestQueue", 10, true)
	assert.NoError(t, err)

	_, err = consumer.GetBatch("ConsumerTestQueue", 10, false)
	assert.NoError(t, err)

	_, err = consumer.GetBatch("ConsumerTestQueue", -1, false)
	assert.Error(t, err)
}

func TestCreateConsumerAndPublisher(t *testing.T) {
	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	publisher, err := publisher.NewPublisher(Seasoning, channelPool, nil)
	assert.NoError(t, err)
	assert.NotNil(t, publisher)

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, err := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err)
	assert.NotNil(t, consumer)
}

func TestCreateConsumerAndUncleanShutdown(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	publisher, err := publisher.NewPublisher(Seasoning, channelPool, nil)
	assert.NoError(t, err)
	assert.NotNil(t, publisher)

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, err := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err)
	assert.NotNil(t, consumer)

	channelPool.Shutdown()
ErrorLoop:
	for {
		select {
		case notice := <-publisher.Notifications():
			fmt.Print(notice.ToString())
		case err := <-consumer.Errors():
			fmt.Printf("%s\r\n", err)
		case err := <-channelPool.Errors():
			fmt.Printf("%s\r\n", err)
		default:
			break ErrorLoop
		}
	}
}

func TestPublishAndConsume(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	channelPool, err := pools.NewChannelPool(Seasoning.PoolConfig, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	publisher, err := publisher.NewPublisher(Seasoning, channelPool, nil)
	assert.NoError(t, err)
	assert.NotNil(t, publisher)

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, err := consumer.NewConsumerFromConfig(consumerConfig, channelPool)
	assert.NoError(t, err)
	assert.NotNil(t, consumer)

	publisher.StartAutoPublish(false)

	publisher.QueueLetter(utils.CreateMockRandomLetter("ConsumerTestQueue"))

	err = consumer.StartConsuming()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Second)

ConsumeMessages:
	for {
		select {
		case <-ctx.Done():
			t.Log("\r\nContextTimeout\r\n")
			break ConsumeMessages
		case notice := <-publisher.Notifications():
			t.Logf("UpperLoop: %s\r\n", notice.ToString())
		case message := <-consumer.Messages():
			fmt.Printf("Message Received: %s\r\n", string(message.Body))
			if err := consumer.StopConsuming(false, true); err != nil {
				t.Error(err)
			}
			break ConsumeMessages
		case err := <-consumer.Errors():
			assert.NoError(t, err)
		case err := <-channelPool.Errors():
			assert.NoError(t, err)
		default:
			time.Sleep(100 * time.Millisecond)
			break
		}
	}

	publisher.StopAutoPublish()
	channelPool.Shutdown()

ErrorLoop:
	for {
		select {
		case notice := <-publisher.Notifications():
			fmt.Printf("LowerLoop: %s", notice.ToString())
		case err := <-consumer.Errors():
			fmt.Printf("%s\r\n", err)
		case err := <-channelPool.Errors():
			fmt.Printf("%s\r\n", err)
		default:
			break ErrorLoop
		}
	}

	cancel()
}

func TestPublishAndConsumeMany(t *testing.T) {

	t.Logf("%s: Benchmark started...", time.Now())

	messageCount := 200000

	publisher, _ := publisher.NewPublisher(Seasoning, ChannelPool, nil)

	consumerConfig, ok := Seasoning.ConsumerConfigs["TurboCookedRabbitConsumer-AutoAck"]
	assert.True(t, ok)

	consumer, _ := consumer.NewConsumerFromConfig(consumerConfig, ChannelPool)

	publisher.StartAutoPublish(false)

	createAndQueueLetters(t, messageCount, publisher)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(2*time.Minute))
	defer cancel()

	messagesPublished, messagesFailedToPublish := monitorPublishing(ctx, t, publisher, messageCount)

	if err := consumer.StartConsuming(); err != nil {

		t.Error(err)
	}

	consumeMessages(ctx, t, consumer, messageCount, messagesPublished, messagesFailedToPublish)

	publisher.StopAutoPublish()
	if err := consumer.StopConsuming(false, true); err != nil {
		t.Error(err)
	}

}

func createAndQueueLetters(t *testing.T, messageCount int, publisher *publisher.Publisher) {

	letters := make([]*models.Letter, messageCount)
	for i := 0; i < messageCount; i++ {
		letters[i] = utils.CreateMockRandomLetter("ConsumerTestQueue")
	}

	for i := 0; i < len(letters); i++ {
		publisher.QueueLetter(letters[i])
	}
}

func monitorPublishing(ctx context.Context, t *testing.T, publisher *publisher.Publisher, messageCount int) (int, int) {

	messagesPublished := 0
	messagesFailedToPublish := 0

MonitorMessages:
	for {
		select {
		case <-ctx.Done():
			t.Logf("%s\r\nContextTimeout\r\n", time.Now())
			break MonitorMessages
		case notice := <-publisher.Notifications():
			if notice.Success {
				messagesPublished++
			} else {
				messagesFailedToPublish++
				t.Logf("%s: Message [ID: %d] failed to publish.", time.Now(), notice.LetterID)
			}

			if messagesPublished+messagesFailedToPublish == messageCount {
				break MonitorMessages
			}
		}
	}

	return messagesPublished, messagesFailedToPublish
}

func consumeMessages(
	ctx context.Context,
	t *testing.T,
	consumer *consumer.Consumer,
	messageCount int,
	messagesPublished int,
	messagesFailedToPublish int) {

	messagesReceived := 0
	consumerErrors := 0
	channelPoolErrors := 0

	startTime := time.Now()
ConsumeMessages:
	for {
		select {
		case <-ctx.Done():
			t.Logf("%s\r\nContextTimeout\r\n", time.Now())
			break ConsumeMessages
		case err := <-consumer.Errors():
			if err != nil {
				t.Log(err)
			}
			consumerErrors++
		case <-consumer.Messages():
			messagesReceived++
		case err := <-ChannelPool.Errors():
			if err != nil {
				t.Log(err)
			}
			channelPoolErrors++
		default:
			time.Sleep(100 * time.Nanosecond)
			break
		}

		if messagesReceived+messagesFailedToPublish >= messageCount {
			break ConsumeMessages
		}
	}
	elapsedTime := time.Since(startTime)

	assert.Equal(t, messageCount, messagesReceived+messagesFailedToPublish)
	t.Logf("%s: Test finished, elapsed time: %f s", time.Now(), elapsedTime.Seconds())
	t.Logf("%s: Consumer Rate: %f msgs/s", time.Now(), (float64(messagesReceived) / elapsedTime.Seconds()))
	t.Logf("%s: Channel Pool Errors: %d\r\n", time.Now(), channelPoolErrors)
	t.Logf("%s: Consumer Errors: %d\r\n", time.Now(), consumerErrors)
	t.Logf("%s: Messages Published: %d\r\n", time.Now(), messagesPublished)
	t.Logf("%s: Messages Failed to Publish: %d\r\n", time.Now(), messagesFailedToPublish)
	t.Logf("%s: Messages Received: %d\r\n", time.Now(), messagesReceived)
}
