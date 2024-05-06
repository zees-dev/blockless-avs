#!/bin/bash

RPC_URL=http://localhost:8545
# address: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (anvil - account 0)
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
# At this point we are in tests/anvil
cd "$parent_path"

# start an empty anvil chain in the background and dump its state to a json file upon exit
mkdir -p snapshots
anvil --dump-state snapshots/eigenlayer-deployed-anvil-state.json &

cd ../contracts/lib/eigenlayer-middleware/lib/eigenlayer-contracts
# M2_Deploy_From_Scratch.s.sol prepends "script/configs/devnet/" to the configFile passed as input (M2_deploy_from_scratch.anvil.config.json)
forge script script/deploy/M2_Deploy_From_Scratch.s.sol --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast --sig "run(string memory configFile)" -- M2_deploy_from_scratch.anvil.config.json

# move the output file - which contains deployed addresses to main project
mkdir -p ../../../../script/output/31337/
mv script/output/M2_from_scratch_deployment_data.json ../../../../script/output/31337/eigenlayer_deployment_output.json

# kill anvil to save its state
pkill anvil
