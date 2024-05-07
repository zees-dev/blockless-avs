#!/bin/bash

set -e  # exit on failure
set -m  # enable job control

RPC_URL=http://localhost:8545
# address: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (anvil - account 0)
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
# retrieved from config-files/keys/test.ecdsa.key.json
# OPERATOR_ADDRESS=$(jq '.address' config-files/keys/test.ecdsa.key.json | tr -d '"')
OPERATOR_ADDRESS=0x860B6912C2d0337ef05bbC89b0C2CB6CbAEAB4A5

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

# pre-deployed holesky contracts
DEPLOYMENT_FILE='../../contracts/lib/eigenlayer-middleware/lib/eigenlayer-contracts/script/output/holesky/M2_deploy_from_scratch.holesky.config.json'
CHAIN_ID=$(jq -r '.chainInfo.chainId' $DEPLOYMENT_FILE)

# copy the eigenlayer_deployment_output.json for holesky from eigenlayer-contracts repo to current project
rm -rf ../../contracts/script/output/$CHAIN_ID/
mkdir -p ../../contracts/script/output/$CHAIN_ID/
cp $DEPLOYMENT_FILE ../../contracts/script/output/$CHAIN_ID/eigenlayer_deployment_output.json 

# start a forked (holesky) anvil chain at specific block number which has eigenlayer contracts deployed;
# TODO: may not be able to specify block-no. since state retrieval may not be possible if RPC is not archive node
anvil --fork-url https://ethereum-holesky-rpc.publicnode.com &

cd ../../contracts

# issue operator some ether
# FIXME: this seems to be required before running next script to advance state of chain
sleep 5 # wait for forked anvil to start with forked state at latest block from fork-url
cast send $OPERATOR_ADDRESS --value 10ether --private-key $PRIVATE_KEY

forge script script/holesky/IncredibleSquaringDeployer.s.sol --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast -v
# # save the block-number in the genesis file which we also need to restart the anvil chain at the correct block
# # otherwise the indexRegistry has a quorumUpdate at a high block number, and when we restart a clean anvil (without genesis.json) file
# # it starts at block 0, and so calling getOperatorListAtBlockNumber reverts because it thinks there are no quorums registered at block 0
# # EDIT: this doesn't actually work... since we can't both load a state and a genesis.json file... see https://github.com/foundry-rs/foundry/issues/6679
# # will keep here in case this PR ever gets merged.
# GENESIS_FILE=$parent_path/genesis.json
# TMP_GENESIS_FILE=$parent_path/genesis.json.tmp
# jq '.number = "'$(cast block-number)'"' $GENESIS_FILE > $TMP_GENESIS_FILE
# mv $TMP_GENESIS_FILE $GENESIS_FILE

# kill anvil to save its state
# pkill anvil

# bring anvil to foreground
# jobs
echo "running holesky forked anvil at port 8545..."
fg
