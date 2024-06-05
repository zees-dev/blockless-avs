package chainio

import (
	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"github.com/zees-dev/blockless-avs/core"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/Layr-Labs/eigensdk-go/chainio/clients/eth"
	sdklogging "github.com/Layr-Labs/eigensdk-go/logging"
)

type AvsSubscriberer interface {
	SubscribeToOracleUpdateResponses(oracleUpdateChan chan *csavs.ContractBlocklessAVSOracleUpdate) event.Subscription
}

// Subscribers use a ws connection instead of http connection like Readers
// kind of stupid that the geth client doesn't have a unified interface for both...
// it takes a single url, so the bindings, even though they have watcher functions, those can't be used
// with the http connection... seems very very stupid. Am I missing something?
type AvsSubscriber struct {
	AvsContractBindings *AvsManagersBindings
	logger              sdklogging.Logger
}

func BuildAvsSubscriberFromConfig(config *core.Config) (*AvsSubscriber, error) {
	return BuildAvsSubscriber(
		config.BlocklessAVSRegistryCoordinatorAddr,
		config.OperatorStateRetrieverAddr,
		*config.EthWsClient,
		config.Logger,
	)
}

func BuildAvsSubscriber(registryCoordinatorAddr, blsOperatorStateRetrieverAddr gethcommon.Address, ethclient eth.Client, logger sdklogging.Logger) (*AvsSubscriber, error) {
	avsContractBindings, err := NewAvsManagersBindings(registryCoordinatorAddr, blsOperatorStateRetrieverAddr, ethclient, logger)
	if err != nil {
		logger.Errorf("Failed to create contract bindings", "err", err)
		return nil, err
	}
	return NewAvsSubscriber(avsContractBindings, logger), nil
}

func NewAvsSubscriber(avsContractBindings *AvsManagersBindings, logger sdklogging.Logger) *AvsSubscriber {
	return &AvsSubscriber{
		AvsContractBindings: avsContractBindings,
		logger:              logger,
	}
}

func (s *AvsSubscriber) SubscribeToOracleUpdateResponses(oracleUpdateChan chan *csavs.ContractBlocklessAVSOracleUpdate) event.Subscription {
	sub, err := s.AvsContractBindings.ServiceManager.WatchOracleUpdate(
		&bind.WatchOpts{}, oracleUpdateChan,
	)
	if err != nil {
		s.logger.Error("Failed to subscribe to ORacleUpdate events", "err", err)
	}
	s.logger.Infof("Subscribed to OracleUpdate events")
	return sub
}
