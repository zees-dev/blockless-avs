#!/bin/bash

# cd to the directory of this script so that this can be run from anywhere
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path"

anvil --load-state ./snapshots/avs-and-eigenlayer-deployed-anvil-state.json
