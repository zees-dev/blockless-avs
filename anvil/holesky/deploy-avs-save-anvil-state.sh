#!/bin/bash

set -e  # exit on failure
set -m  # enable job control

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"


RPC_URL=http://localhost:8545
# HOLESKY_URL=https://ethereum-holesky-rpc.publicnode.com
HOLESKY_URL=https://holesky.infura.io/v3/8792dc3bbc3743f6b884807fb6a22525
# address: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (anvil - account 0)
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# pre-deployed holesky contracts
DEPLOYMENT_FILE='../../contracts/lib/eigenlayer-middleware/lib/eigenlayer-contracts/script/output/holesky/M2_deploy_from_scratch.holesky.config.json'
CHAIN_ID=$(jq -r '.chainInfo.chainId' $DEPLOYMENT_FILE)

# copy the eigenlayer_deployment_output.json for holesky from eigenlayer-contracts repo to current project
rm -rf ../../contracts/script/output/$CHAIN_ID/
mkdir -p ../../contracts/script/output/$CHAIN_ID/
cp $DEPLOYMENT_FILE ../../contracts/script/output/$CHAIN_ID/eigenlayer_deployment_output.json 

# start a forked (holesky) anvil chain at specific block number which has eigenlayer contracts deployed;
# FIXME: this seems to be required before running next script to advance state of chain
anvil --fork-url $HOLESKY_URL &
sleep 5 # wait for forked anvil to start with forked state at latest block from fork-url

cd ../../contracts
forge script script/holesky/BlocklessAVSDeployer.s.sol --rpc-url $RPC_URL  --private-key $PRIVATE_KEY --broadcast -v

# bring anvil to foreground
# jobs
echo "running holesky forked anvil at port 8545..."
fg
