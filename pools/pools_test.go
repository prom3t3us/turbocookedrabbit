package pools_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/houseofcat/turbocookedrabbit/models"
	"github.com/houseofcat/turbocookedrabbit/pools"
	"github.com/houseofcat/turbocookedrabbit/utils"
	"github.com/stretchr/testify/assert"
)

var Seasoning *models.RabbitSeasoning

func TestMain(m *testing.M) { // Load Configuration On Startup
	var err error
	Seasoning, err = utils.ConvertJSONFileToConfig("poolseasoning.json")
	if err != nil {
		return
	}
	os.Exit(m.Run())
}

func TestCreateConnectionPool(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	Seasoning.Pools.ConnectionCount = 10
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	timeStart := time.Now()

	if !connectionPool.Initialized {
		connectionPool.Initialize()
	}

	elapsed := time.Since(timeStart)
	fmt.Printf("Created %d connection(s) finished in %s.\r\n", connectionPool.ConnectionCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())

	connectionPool.FlushErrors()

	connectionPool.Shutdown()
}

func TestCreateConnectionPoolAndShutdown(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	Seasoning.Pools.ConnectionCount = 12
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !connectionPool.Initialized {
		connectionPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())

	timeStart = time.Now()
	connectionPool.Shutdown()
	elapsed = time.Since(timeStart)

	fmt.Printf("Shutdown %d connection(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), elapsed)
	assert.Equal(t, int64(0), connectionPool.ConnectionCount())

	connectionPool.FlushErrors()
	connectionPool.Shutdown()
}

func TestGetConnectionAfterShutdown(t *testing.T) {
	Seasoning.Pools.ConnectionCount = 24
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !connectionPool.Initialized {
		connectionPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())

	connectionPool.FlushErrors()

	connectionCount := connectionPool.ConnectionCount()
	timeStart = time.Now()
	connectionPool.Shutdown()
	elapsed = time.Since(timeStart)

	fmt.Printf("Shutdown %d connection(s). Finished in %s.\r\n", connectionCount, elapsed)
	assert.Equal(t, int64(0), connectionPool.ConnectionCount())

	connectionPool.FlushErrors()

	connHost, err := connectionPool.GetConnection()
	assert.Error(t, err)
	assert.Nil(t, connHost)

	connectionPool.Shutdown()
}

func TestCreateChannelPool(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.
	Seasoning.Pools.ConnectionCount = 10
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	channelPool, err := pools.NewChannelPool(Seasoning, connectionPool, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !channelPool.Initialized {
		channelPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Created %d channel(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), channelPool.ChannelCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())
	assert.Equal(t, Seasoning.Pools.ChannelCount, channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()

	connectionPool.Shutdown()
	channelPool.Shutdown()
}

func TestCreateChannelPoolAndShutdown(t *testing.T) {
	Seasoning.Pools.ConnectionCount = 10
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	channelPool, err := pools.NewChannelPool(Seasoning, connectionPool, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !channelPool.Initialized {
		channelPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Created %d channel(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), channelPool.ChannelCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())
	assert.Equal(t, Seasoning.Pools.ChannelCount, channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()

	channelCount := channelPool.ChannelCount()
	timeStart = time.Now()
	channelPool.Shutdown()
	elapsed = time.Since(timeStart)

	fmt.Printf("Shutdown %d channel(s). Finished in %s.\r\n", channelCount, elapsed)
	assert.Equal(t, int64(0), channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()
	connectionPool.Shutdown()
}

func TestGetChannelAfterShutdown(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	Seasoning.Pools.ConnectionCount = 10
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	channelPool, err := pools.NewChannelPool(Seasoning, connectionPool, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !channelPool.Initialized {
		channelPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Created %d channel(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), channelPool.ChannelCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())
	assert.Equal(t, Seasoning.Pools.ChannelCount, channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()

	channelCount := channelPool.ChannelCount()
	timeStart = time.Now()
	channelPool.Shutdown()
	elapsed = time.Since(timeStart)

	fmt.Printf("Shutdown %d channel(s). Finished in %s.\r\n", channelCount, elapsed)
	assert.Equal(t, int64(0), channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()

	channelHost, err := channelPool.GetChannel()
	assert.Error(t, err)
	assert.Nil(t, channelHost)
}

func TestGetChannelAfterKillingConnectionPool(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	Seasoning.Pools.ConnectionCount = 1
	Seasoning.Pools.ChannelCount = 2
	connectionPool, err := pools.NewConnectionPool(Seasoning, false)
	assert.NoError(t, err)

	channelPool, err := pools.NewChannelPool(Seasoning, connectionPool, false)
	assert.NoError(t, err)

	timeStart := time.Now()
	if !channelPool.Initialized {
		channelPool.Initialize()
	}
	elapsed := time.Since(timeStart)

	fmt.Printf("Created %d connection(s). Created %d channel(s). Finished in %s.\r\n", connectionPool.ConnectionCount(), channelPool.ChannelCount(), elapsed)
	assert.Equal(t, Seasoning.Pools.ConnectionCount, connectionPool.ConnectionCount())
	assert.Equal(t, Seasoning.Pools.ChannelCount, channelPool.ChannelCount())

	connectionPool.FlushErrors()
	channelPool.FlushErrors()

	connectionPool.Shutdown()

	chanHost, err := channelPool.GetChannel()
	assert.Nil(t, chanHost)
	assert.Error(t, err)

	channelPool.Shutdown()
}

func TestCreateChannelPoolSimple(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.

	Seasoning.Pools.ConnectionCount = 1
	Seasoning.Pools.ChannelCount = 2

	channelPool, err := pools.NewChannelPool(Seasoning, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()

	chanHost, err := channelPool.GetChannel()
	assert.NotNil(t, chanHost)
	assert.NoError(t, err)

	channelPool.Shutdown()
}

func TestGetChannelAfterKillingChannelPool(t *testing.T) {
	defer leaktest.Check(t)() // Fail on leaked goroutines.
	Seasoning.Pools.ConnectionCount = 1
	Seasoning.Pools.ChannelCount = 2

	channelPool, err := pools.NewChannelPool(Seasoning, nil, true)
	assert.NoError(t, err)

	channelPool.FlushErrors()
	channelPool.Shutdown()

	chanHost, err := channelPool.GetChannel()
	assert.Nil(t, chanHost)
	assert.Error(t, err)
}