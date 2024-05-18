#!/bin/bash

set -e  # exit on failure

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"


# retrieved from operator_keys/blockless.ecdsa.key.json
# OPERATOR_ADDRESS=$(sed 's/^/0x/' <<< $(jq '.address' ../../operator_keys/blockless.ecdsa.key.json | tr -d '"'))
# OPERATOR_ADDRESS=0xb932aafe9639f0508ab37eeb63564de6e53bd010
PRIVATE_KEY=$PRIVATE_KEY
RPC_URL=https://ethereum-holesky-rpc.publicnode.com

# pre-deployed holesky contracts
DEPLOYMENT_FILE='../../contracts/lib/eigenlayer-middleware/lib/eigenlayer-contracts/script/output/holesky/M2_deploy_from_scratch.holesky.config.json'
CHAIN_ID=$(jq -r '.chainInfo.chainId' $DEPLOYMENT_FILE)

# copy the eigenlayer_deployment_output.json for holesky from eigenlayer-contracts repo to current project
rm -rf ../../contracts/script/output/$CHAIN_ID/
mkdir -p ../../contracts/script/output/$CHAIN_ID/
cp $DEPLOYMENT_FILE ../../contracts/script/output/$CHAIN_ID/eigenlayer_deployment_output.json 

cd ../../contracts
forge script script/holesky/BlocklessAVSDeployer.s.sol --rpc-url $RPC_URL  --private-key $PRIVATE_KEY --broadcast -v
