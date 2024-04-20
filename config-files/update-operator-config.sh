#!/bin/bash

OPERATOR_CONFIG_FILE=./operator.anvil.yaml
CHAINID=$(cast chain-id)
OUTPUT_DEPLOYMENT_FILE=../contracts/script/output/$CHAINID/credible_squaring_avs_deployment_output.json

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# Extract addresses from JSON
REGISTRY_COORDINATOR=$(jq -r '.addresses.registryCoordinator' $OUTPUT_DEPLOYMENT_FILE)
OPERATOR_STATE_RETRIEVER=$(jq -r '.addresses.operatorStateRetriever' $OUTPUT_DEPLOYMENT_FILE)
TOKEN_STRATEGY=$(jq -r '.addresses.erc20MockStrategy' $OUTPUT_DEPLOYMENT_FILE)

# Update YAML file (assume sed running on MacOS)
sed -i '' "s|avs_registry_coordinator_address: .*|avs_registry_coordinator_address: $REGISTRY_COORDINATOR # registryCoordinator|" $OPERATOR_CONFIG_FILE
sed -i '' "s|operator_state_retriever_address: .*|operator_state_retriever_address: $OPERATOR_STATE_RETRIEVER # operatorStateRetriever|" $OPERATOR_CONFIG_FILE
sed -i '' "s|token_strategy_addr: .*|token_strategy_addr: $TOKEN_STRATEGY # erc20MockStrategy|" $OPERATOR_CONFIG_FILE
