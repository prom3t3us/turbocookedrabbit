package services

import (
	"errors"
	"os"
	"sync/atomic"
	"time"

	"github.com/houseofcat/turbocookedrabbit/consumer"
	"github.com/houseofcat/turbocookedrabbit/models"
	"github.com/houseofcat/turbocookedrabbit/pools"
	"github.com/houseofcat/turbocookedrabbit/publisher"
	"github.com/houseofcat/turbocookedrabbit/topology"
)

// RabbitService is the struct for containing RabbitMQ management.
type RabbitService struct {
	Config               *models.RabbitSeasoning
	ChannelPool          *pools.ChannelPool
	Topologer            *topology.Topologer
	Publisher            *publisher.Publisher
	centralErr           chan error
	consumers            map[string]*consumer.Consumer
	stopServiceSignal    chan bool
	stop                 bool
	retryCount           uint32
	letterCount          uint64
	monitorSleepInterval time.Duration
}

// NewRabbitService creates everything you need for a RabbitMQ communication service.
func NewRabbitService(config *models.RabbitSeasoning) (*RabbitService, error) {

	channelPool, err := pools.NewChannelPool(config.PoolConfig, nil, true)
	if err != nil {
		return nil, err
	}

	publisher, err := publisher.NewPublisher(config, channelPool, nil)
	if err != nil {
		return nil, err
	}

	topologer, err := topology.NewTopologer(channelPool)
	if err != nil {
		return nil, err
	}

	rs := &RabbitService{
		ChannelPool:          channelPool,
		Config:               config,
		Publisher:            publisher,
		Topologer:            topologer,
		centralErr:           make(chan error, config.ServiceConfig.ErrorBuffer),
		stopServiceSignal:    make(chan bool, 1),
		consumers:            make(map[string]*consumer.Consumer),
		retryCount:           10,
		monitorSleepInterval: time.Duration(3) * time.Second,
	}

	err = rs.CreateConsumers(config.ConsumerConfigs)
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// CreateConsumers takes a config from the Config and builds all the consumers (errors if config is missing).
func (rs *RabbitService) CreateConsumers(consumerConfigs map[string]*models.ConsumerConfig) error {

	for consumerName, consumerConfig := range consumerConfigs {

		consumer, err := consumer.NewConsumerFromConfig(consumerConfig, rs.ChannelPool)
		if err != nil {
			return err
		}

		hostName, err := os.Hostname()
		if err == nil {
			consumer.ConsumerName = hostName + "-" + consumer.ConsumerName
		}

		rs.consumers[consumerName] = consumer
	}

	return nil
}

// CreateConsumerFromConfig takes a config from the Config map and builds a consumer (errors if config is missing).
func (rs *RabbitService) CreateConsumerFromConfig(consumerName string) error {
	if consumerConfig, ok := rs.Config.ConsumerConfigs[consumerName]; ok {
		consumer, err := consumer.NewConsumerFromConfig(consumerConfig, rs.ChannelPool)
		if err != nil {
			return err
		}
		rs.consumers[consumerName] = consumer
		return nil
	}
	return nil
}

// PublishWithRetry tries to publish with a retry mechanism.
func (rs *RabbitService) PublishWithRetry(body []byte, exchangeName, routingKey string) error {
	if body == nil || (exchangeName == "" && routingKey == "") {
		return errors.New("can't have a nil body or an empty exchangename with empty routing key")
	}

	currentCount := atomic.LoadUint64(&rs.letterCount)
	atomic.AddUint64(&rs.letterCount, 1)

	rs.Publisher.PublishWithRetry(
		&models.Letter{
			LetterID:   currentCount,
			RetryCount: rs.retryCount,
			Body:       body,
			Envelope: &models.Envelope{
				Exchange:    exchangeName,
				RoutingKey:  routingKey,
				ContentType: "application/json",
				Mandatory:   false,
				Immediate:   false,
			},
		})

	return nil
}

// Publish tries to publish directly without retry.
func (rs *RabbitService) Publish(body []byte, exchangeName, routingKey string) error {
	if body == nil || (exchangeName == "" && routingKey == "") {
		return errors.New("can't have a nil body or an empty exchangename with empty routing key")
	}

	currentCount := atomic.LoadUint64(&rs.letterCount)
	atomic.AddUint64(&rs.letterCount, 1)

	rs.Publisher.Publish(
		&models.Letter{
			LetterID:   currentCount,
			RetryCount: rs.retryCount,
			Body:       body,
			Envelope: &models.Envelope{
				Exchange:    exchangeName,
				RoutingKey:  routingKey,
				ContentType: "application/json",
				Mandatory:   false,
				Immediate:   false,
			},
		})

	return nil
}

// StartService gets all the background internals and logging/monitoring started.
func (rs *RabbitService) StartService() {

	// Start the background monitors and logging.
	rs.collectPublisherErrors()
	rs.collectChannelPoolErrors()
	rs.collectConsumerErrors()

	// Start the AutoPublisher
	rs.Publisher.StartAutoPublish(false)
}

func (rs *RabbitService) monitorStopService() {
	go func() {
	MonitorLoop:
		for {
			select {
			case <-rs.stopServiceSignal:
				rs.stop = true
				break MonitorLoop
			default:
				time.Sleep(rs.monitorSleepInterval)
				break
			}
		}
	}()
}

func (rs *RabbitService) collectPublisherErrors() {
	go func() {
	MonitorLoop:
		for {
			if rs.stop {
				break MonitorLoop
			}

			select {
			case notification := <-rs.Publisher.Notifications():
				if notification.Success {

				} else {

				}
			default:
				time.Sleep(rs.monitorSleepInterval)
				break
			}
		}
	}()
}

func (rs *RabbitService) collectChannelPoolErrors() {
	go func() {
	MonitorLoop:
		for {
			if rs.stop {
				break MonitorLoop
			}

			select {
			case err := <-rs.ChannelPool.Errors():
				rs.centralErr <- err
			default:
				time.Sleep(rs.monitorSleepInterval)
				break
			}
		}
	}()
}

func (rs *RabbitService) collectConsumerErrors() {
	go func() {

	MonitorLoop:
		for {

			for _, consumer := range rs.consumers {
			IndividualConsumerLoop:
				for {
					if rs.stop {
						break MonitorLoop
					}

					select {
					case err := <-consumer.Errors():
						rs.centralErr <- err
					default:
						break IndividualConsumerLoop
					}
				}
			}

			time.Sleep(rs.monitorSleepInterval)
		}
	}()
}

// GetConsumer allows you to get the individual consumer.
func (rs *RabbitService) GetConsumer(consumerName string) (*consumer.Consumer, error) {
	if consumer, ok := rs.consumers[consumerName]; ok {
		return consumer, nil
	}

	return nil, errors.New("consumer was not found")
}

// StopService stops the AutoPublisher, Consumer, and Monitoring.
func (rs *RabbitService) StopService() {
	rs.Publisher.StopAutoPublish()

	time.Sleep(1 * time.Second)
	rs.stopServiceSignal <- true
	time.Sleep(1 * time.Second)
}

// Shutdown stops the service and shuts down the ChannelPool.
func (rs *RabbitService) Shutdown(stopConsumers bool) {
	rs.StopService()

	if stopConsumers {
		for _, consumer := range rs.consumers {
			err := consumer.StopConsuming(true, true)
			if err != nil {
				rs.centralErr <- err
			}
		}
	}

	rs.ChannelPool.Shutdown()
}