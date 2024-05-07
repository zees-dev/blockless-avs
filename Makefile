############################# HELP MESSAGE #############################
# Make sure the help command stays first, so that it's printed by default when `make` is called without arguments
.PHONY: help tests
help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# account (9) on anvil
AGGREGATOR_ECDSA_PRIV_KEY=0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6
# account (2) on anvil
CHALLENGER_ECDSA_PRIV_KEY=0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a

CHAINID=31337
# check in contracts/script/output/${CHAINID}/credible_squaring_avs_deployment_output.json
DEPLOYMENT_FILES_DIR=contracts/script/output/${CHAINID}

HOLESKY_CHAIN_ID=17000
HOLESKY_DEPLOYMENT_FILES_DIR=contracts/script/output/${HOLESKY_CHAIN_ID}

-----------------------------: ## 

clean:
	rm -rf anvil/snapshots
	cd contracts && forge clean && rm -rf cache && rm -rf script/output

clean-node:
	rm -rf ./node/peer-db/
	rm -rf ./node/function-db/

___CONTRACTS___: ## 
## Deploy all contracts, start anvil with deployed contracts
start-anvil-all-deployed:
	echo "Deploying eigenlayer contracts..."
	make deploy-eigenlayer-contracts-to-anvil-and-save-state
	echo "Deploying AVS contracts..."
	make deploy-avs-contracts-to-anvil-and-save-state
	echo "Starting anvil with EL and AVS contracts deployed..."
	make start-anvil-chain-with-el-and-avs-deployed

build-contracts: ## builds all contracts
	cd contracts && forge build

## Deploy eigenlayer contracts
## Save snapshot of anvil state and outputs contract addresses to json file
deploy-eigenlayer-contracts-to-anvil-and-save-state:
	./anvil/deploy-eigenlayer-save-anvil-state.sh

## Deploy AVS contracts
## Save snapshot of anvil state and outputs contract addresses to json file
deploy-avs-contracts-to-anvil-and-save-state:
	./anvil/deploy-avs-save-anvil-state.sh

## starts anvil from a saved state file (with el and avs contracts deployed)
## note: we may not need this, as we can just deploy the contracts, start the chain, advance blocks
start-anvil-chain-with-el-and-avs-deployed:
	./anvil/start-anvil-chain-with-el-and-avs-deployed.sh

## Generate contract go bindings
bindings:
	cd contracts && ./generate-go-bindings.sh

## Deploy all contracts on holesky forked local net, start anvil with deployed contracts
holesky-start-anvil-all-deployed:
	echo "Starting anvil; deploying AVS contracts..."
	./anvil/holesky/deploy-avs-save-anvil-state.sh

___DOCKER___: ## 
docker-build-and-publish-images: ## builds and publishes operator and aggregator docker images using Ko
	KO_DOCKER_REPO=ghcr.io/layr-labs/incredible-squaring ko build aggregator/cmd/main.go --preserve-import-paths
	KO_DOCKER_REPO=ghcr.io/layr-labs/incredible-squaring ko build operator/cmd/main.go --preserve-import-paths
docker-start-everything: docker-build-and-publish-images ## starts aggregator and operator docker containers
	docker compose pull && docker compose up

__OPERATOR__: ## 
cli-setup-operator: ## registers operator with eigenlayer and avs
	echo "Updating operator.anvil.yaml config..."
	make cli-update-operator-config
	echo "Registering operator with eigenlayer"
	make cli-register-operator-with-eigenlayer
	echo "Depositing into mocktoken strategy"
	make cli-deposit-into-mocktoken-strategy
	echo "Registering operator with avs"
	make cli-register-operator-with-avs
	make cli-print-operator-status

holesky-cli-setup-operator: ## registers operator with eigenlayer and avs
	echo "Updating operator.anvil.yaml config..."
	make cli-update-operator-config
	# NOTE: operator ecdsa account already registered with eigenlayer on testnet
	# echo "Registering operator with eigenlayer"
	# make cli-register-operator-with-eigenlayer
	echo "Depositing into mocktoken strategy"
	make cli-deposit-into-mocktoken-strategy
	echo "Registering operator with avs"
	make cli-register-operator-with-avs
	make cli-print-operator-status

cli-update-operator-config: ## updates operator.anvil.yaml config from generated/deployed contract addresses
	./config-files/update-operator-config.sh

cli-register-operator-with-eigenlayer:
	go run cli/*.go register-operator-with-eigenlayer --config config-files/operator.anvil.yaml

cli-deposit-into-mocktoken-strategy:
	go run cli/*.go deposit-into-strategy --config config-files/operator.anvil.yaml --amount 100

cli-register-operator-with-avs:
	go run cli/*.go register-operator-with-avs --config config-files/operator.anvil.yaml

cli-deregister-operator-with-avs:
	go run cli/*.go deregister-operator-with-avs --config config-files/operator.anvil.yaml

cli-print-operator-status:
	go run cli/*.go print-operator-status --config config-files/operator.anvil.yaml

cli-run-avs:
	go run cli/*.go run-avs

-----------------------------: ## 
# We pipe all zapper logs through https://github.com/maoueh/zap-pretty so make sure to install it
# TODO: piping to zap-pretty only works when zapper environment is set to production, unsure why
____OFFCHAIN_SOFTWARE___: ## 
start-aggregator: ##
	go run aggregator/cmd/main.go --config config-files/aggregator.yaml \
		--credible-squaring-deployment ${DEPLOYMENT_FILES_DIR}/credible_squaring_avs_deployment_output.json \
		--ecdsa-private-key ${AGGREGATOR_ECDSA_PRIV_KEY} \
		2>&1 | zap-pretty

holesky-start-aggregator:
	go run aggregator/cmd/main.go --config config-files/aggregator.yaml \
		--credible-squaring-deployment ${HOLESKY_DEPLOYMENT_FILES_DIR}/credible_squaring_avs_deployment_output.json \
		--ecdsa-private-key ${AGGREGATOR_ECDSA_PRIV_KEY} \
		2>&1 | zap-pretty

start-operator: ## 
	go run operator/cmd/main.go --config config-files/operator.anvil.yaml \
		2>&1 | zap-pretty

start-challenger: ## 
	go run challenger/cmd/main.go --config config-files/challenger.yaml \
		--credible-squaring-deployment ${DEPLOYMENT_FILES_DIR}/credible_squaring_avs_deployment_output.json \
		--ecdsa-private-key ${CHALLENGER_ECDSA_PRIV_KEY} \
		2>&1 | zap-pretty

run-plugin: ## 
	go run plugin/cmd/main.go --config config-files/operator.anvil.yaml
-----------------------------: ## 
