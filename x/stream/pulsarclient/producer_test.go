package pulsarclient

import (
	"github.com/okex/exchain/x/common"
	"github.com/okex/exchain/x/stream/common/kline"
	"math/rand"
	"os"
	"testing"
	"time"

	appCfg "github.com/okex/exchain/libs/cosmos-sdk/server/config"
	"github.com/okex/exchain/x/backend"
	"github.com/stretchr/testify/require"
	"github.com/okex/exchain/libs/tendermint/libs/log"
)

func TestNewPulsarProducer(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	asyncErrs := make(chan error, 8)
	var err error
	defer func() {
		if len(asyncErrs) != 0 {
			err = <-asyncErrs
		}
		require.Error(t, err)
	}()

	_ = NewPulsarProducer("1.2.3.4:6650", appCfg.DefaultConfig().StreamConfig, logger, &asyncErrs)
	time.Sleep(time.Second * 4)
}

func TestSendMsg(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	asyncErrs := make(chan error, 8)
	go func() {
		for err := range asyncErrs {
			require.NoError(t, err)
		}
	}()
	mp := NewPulsarProducer("localhost:6650", appCfg.DefaultConfig().StreamConfig, logger, &asyncErrs)

	data := kline.NewKlineData()
	data.Height = 9
	_, err := mp.SendAllMsg(data, logger)
	require.NoError(t, err)
	logger.Info("send zero matchResult")

	kline.GetMarketIDMap()["xxb_"+common.NativeToken] = int64(9999)
	results10 := make([]*backend.MatchResult, 0, 10)
	timestamp := time.Now().Unix()
	for i := 0; i < 10; i++ {
		results10 = append(results10, &backend.MatchResult{
			BlockHeight: int64(i),
			Product:     "test_" + common.NativeToken,
			Price:       rand.Float64(),
			Quantity:    rand.Float64(),
			Timestamp:   timestamp,
		})
	}

	data = kline.NewKlineData()
	data.Height = 11
	data.SetMatchResults(results10)
	_, err = mp.SendAllMsg(data, logger)
	if err != nil {
		logger.Info("send 10 matchResult failed")
	}
	require.NoError(t, err)
	logger.Info("send 10 matchResult success")

	results10 = make([]*backend.MatchResult, 0, 10)
	kline.GetMarketIDMap()[common.TestToken+common.NativeToken] = int64(10000)
	for i := 0; i < 10; i++ {
		results10 = append(results10, &backend.MatchResult{
			BlockHeight: int64(i),
			Product:     common.TestToken + common.NativeToken,
			Price:       rand.Float64(),
			Quantity:    rand.Float64(),
			Timestamp:   timestamp,
		})
	}

	data.SetMatchResults(results10)
	_, err = mp.SendAllMsg(data, logger)
	if err != nil {
		logger.Info("send 10 matchResult failed")
	}
	require.NoError(t, err)
	logger.Info("send 10 matchResult success")
}
