// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";

import "@eigenlayer/contracts/permissions/PauserRegistry.sol";
import {IDelegationManager} from "@eigenlayer/contracts/interfaces/IDelegationManager.sol";
import {IAVSDirectory} from "@eigenlayer/contracts/interfaces/IAVSDirectory.sol";
import {IStrategyManager, IStrategy} from "@eigenlayer/contracts/interfaces/IStrategyManager.sol";
import {ISlasher} from "@eigenlayer/contracts/interfaces/ISlasher.sol";
import {StrategyBaseTVLLimits} from "@eigenlayer/contracts/strategies/StrategyBaseTVLLimits.sol";
import "@eigenlayer/test/mocks/EmptyContract.sol";

import "@eigenlayer-middleware/src/RegistryCoordinator.sol" as regcoord;
import {IBLSApkRegistry, IIndexRegistry, IStakeRegistry} from "@eigenlayer-middleware/src/RegistryCoordinator.sol";
import {BLSApkRegistry} from "@eigenlayer-middleware/src/BLSApkRegistry.sol";
import {IndexRegistry} from "@eigenlayer-middleware/src/IndexRegistry.sol";
import {StakeRegistry} from "@eigenlayer-middleware/src/StakeRegistry.sol";
import "@eigenlayer-middleware/src/OperatorStateRetriever.sol";

import {IServiceManager} from "@eigenlayer-middleware/src/interfaces/IServiceManager.sol";
import {BlocklessAVS} from "../../src/BlocklessAVS.sol";
import "../../src/ERC20Mock.sol";

import {Utils} from "../utils/Utils.sol";

import "forge-std/Test.sol";
import "forge-std/Script.sol";
import "forge-std/StdJson.sol";
import "forge-std/console.sol";

// # To deploy and verify our contract
// forge script script/holesky/BlocklessAVSDeployer.s.sol:BlocklessAVSDeployer --rpc-url $RPC_URL  --private-key $PRIVATE_KEY --broadcast -vvvv
contract BlocklessAVSDeployer is Script, Utils {
    // DEPLOYMENT CONSTANTS
    uint256 public constant QUORUM_THRESHOLD_PERCENTAGE = 100;
    uint32 public constant TASK_RESPONSE_WINDOW_BLOCK = 30;
    uint32 public constant TASK_DURATION_BLOCKS = 0;
    // TODO: right now hardcoding these (this address is anvil's default address 9)
    address public constant AGGREGATOR_ADDR = 0xa0Ee7A142d267C1f36714E4a8F75612F20a79720;

    // ERC20 and Strategy: we need to deploy this erc20, create a strategy for it, and whitelist this strategy in the strategymanager

    // TODO: remove these vars after e2e example working; since we are not using mocks - but instead using actual strats
    ERC20Mock public erc20Mock;
    StrategyBaseTVLLimits public erc20MockStrategy;

    // blockless avs contracts
    ProxyAdmin public blocklessAVSProxyAdmin;
    PauserRegistry public blocklessAVSPauserReg;

    regcoord.RegistryCoordinator public registryCoordinator;
    regcoord.IRegistryCoordinator public registryCoordinatorImplementation;

    IBLSApkRegistry public blsApkRegistry;
    IBLSApkRegistry public blsApkRegistryImplementation;

    IIndexRegistry public indexRegistry;
    IIndexRegistry public indexRegistryImplementation;

    IStakeRegistry public stakeRegistry;
    IStakeRegistry public stakeRegistryImplementation;

    OperatorStateRetriever public operatorStateRetriever;

    BlocklessAVS public blocklessAVS;
    IServiceManager public blocklessAVSImplementation;

    function run() external {
        // Eigenlayer contracts
        string memory eigenlayerDeployedContracts = readOutput(
            "eigenlayer_deployment_output"
        );
        IStrategyManager strategyManager = IStrategyManager(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.strategyManager"
            )
        );
        IDelegationManager delegationManager = IDelegationManager(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.delegationManager"
            )
        );
        IAVSDirectory avsDirectory = IAVSDirectory(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.avsDirectory"
            )
        );
        ProxyAdmin eigenLayerProxyAdmin = ProxyAdmin(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.eigenLayerProxyAdmin"
            )
        );
        PauserRegistry eigenLayerPauserReg = PauserRegistry(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.eigenLayerPauserReg"
            )
        );
        StrategyBaseTVLLimits baseStrategyImplementation = StrategyBaseTVLLimits(
                stdJson.readAddress(
                    eigenlayerDeployedContracts,
                    ".addresses.baseStrategyImplementation"
                )
            );

        address blocklessAVSCommunityMultisig = msg.sender;
        address blocklessAVSPauser = msg.sender;

        // reference actual strategies
        // TODO: refactor
        erc20MockStrategy = StrategyBaseTVLLimits(
            stdJson.readAddress(
                eigenlayerDeployedContracts,
                ".addresses.strategies.WETH"
            )
        );
        erc20Mock = ERC20Mock(address(erc20MockStrategy.underlyingToken()));

        vm.startBroadcast();
        _deployBlockessAVSContracts(
            delegationManager,
            avsDirectory,
            erc20MockStrategy,
            blocklessAVSCommunityMultisig,
            blocklessAVSPauser
        );
        vm.stopBroadcast();
    }

    function _deployBlockessAVSContracts(
        IDelegationManager delegationManager,
        IAVSDirectory avsDirectory,
        IStrategy strat,
        address blocklessAVSCommunityMultisig,
        address blocklessAVSPauser
    ) internal {
        // Adding this as a temporary fix to make the rest of the script work with a single strategy
        // since it was originally written to work with an array of strategies
        IStrategy[1] memory deployedStrategyArray = [strat];
        uint numStrategies = deployedStrategyArray.length;

        // deploy proxy admin for ability to upgrade proxy contracts
        blocklessAVSProxyAdmin = new ProxyAdmin();

        // deploy pauser registry
        {
            address[] memory pausers = new address[](2);
            pausers[0] = blocklessAVSPauser;
            pausers[1] = blocklessAVSCommunityMultisig;
            blocklessAVSPauserReg = new PauserRegistry(
                pausers,
                blocklessAVSCommunityMultisig
            );
        }

        EmptyContract emptyContract = new EmptyContract();

        // hard-coded inputs

        /**
         * First, deploy upgradeable proxy contracts that **will point** to the implementations. Since the implementation contracts are
         * not yet deployed, we give these proxies an empty contract as the initial implementation, to act as if they have no code.
         */
        blocklessAVS = BlocklessAVS(
            address(
                new TransparentUpgradeableProxy(
                    address(emptyContract),
                    address(blocklessAVSProxyAdmin),
                    ""
                )
            )
        );
        registryCoordinator = regcoord.RegistryCoordinator(
            address(
                new TransparentUpgradeableProxy(
                    address(emptyContract),
                    address(blocklessAVSProxyAdmin),
                    ""
                )
            )
        );
        blsApkRegistry = IBLSApkRegistry(
            address(
                new TransparentUpgradeableProxy(
                    address(emptyContract),
                    address(blocklessAVSProxyAdmin),
                    ""
                )
            )
        );
        indexRegistry = IIndexRegistry(
            address(
                new TransparentUpgradeableProxy(
                    address(emptyContract),
                    address(blocklessAVSProxyAdmin),
                    ""
                )
            )
        );
        stakeRegistry = IStakeRegistry(
            address(
                new TransparentUpgradeableProxy(
                    address(emptyContract),
                    address(blocklessAVSProxyAdmin),
                    ""
                )
            )
        );

        operatorStateRetriever = new OperatorStateRetriever();

        // Second, deploy the *implementation* contracts, using the *proxy contracts* as inputs
        {
            stakeRegistryImplementation = new StakeRegistry(
                registryCoordinator,
                delegationManager
            );

            blocklessAVSProxyAdmin.upgrade(
                TransparentUpgradeableProxy(payable(address(stakeRegistry))),
                address(stakeRegistryImplementation)
            );

            blsApkRegistryImplementation = new BLSApkRegistry(
                registryCoordinator
            );

            blocklessAVSProxyAdmin.upgrade(
                TransparentUpgradeableProxy(payable(address(blsApkRegistry))),
                address(blsApkRegistryImplementation)
            );

            indexRegistryImplementation = new IndexRegistry(
                registryCoordinator
            );

            blocklessAVSProxyAdmin.upgrade(
                TransparentUpgradeableProxy(payable(address(indexRegistry))),
                address(indexRegistryImplementation)
            );
        }

        registryCoordinatorImplementation = new regcoord.RegistryCoordinator(
            blocklessAVS,
            regcoord.IStakeRegistry(address(stakeRegistry)),
            regcoord.IBLSApkRegistry(address(blsApkRegistry)),
            regcoord.IIndexRegistry(address(indexRegistry))
        );

        {
            uint numQuorums = 1; 

            // for each quorum to setup, we need to define
            // QuorumOperatorSetParam, minimumStakeForQuorum, and strategyParams
            regcoord.IRegistryCoordinator.OperatorSetParam[]
                memory quorumsOperatorSetParams = new regcoord.IRegistryCoordinator.OperatorSetParam[](
                    numQuorums
                );
            for (uint i = 0; i < numQuorums; i++) {
                // hard code these for now
                quorumsOperatorSetParams[i] = regcoord
                    .IRegistryCoordinator
                    .OperatorSetParam({
                        maxOperatorCount: 10000,
                        kickBIPsOfOperatorStake: 15000,
                        kickBIPsOfTotalStake: 100
                    });
            }
            // set to 0 for every quorum
            uint96[] memory quorumsMinimumStake = new uint96[](numQuorums);
            IStakeRegistry.StrategyParams[][]
                memory quorumsStrategyParams = new IStakeRegistry.StrategyParams[][](
                    numQuorums
                );
            for (uint i = 0; i < numQuorums; i++) {
                quorumsStrategyParams[i] = new IStakeRegistry.StrategyParams[](
                    numStrategies
                );
                for (uint j = 0; j < numStrategies; j++) {
                    quorumsStrategyParams[i][j] = IStakeRegistry
                        .StrategyParams({
                            strategy: deployedStrategyArray[j],
                            // setting this to 1 ether since the divisor is also 1 ether
                            // therefore this allows an operator to register with even just 1 token
                            // see https://github.com/Layr-Labs/eigenlayer-middleware/blob/m2-mainnet/src/StakeRegistry.sol#L484
                            //    weight += uint96(sharesAmount * strategyAndMultiplier.multiplier / WEIGHTING_DIVISOR);
                            multiplier: 1 ether
                        });
                }
            }
            blocklessAVSProxyAdmin.upgradeAndCall(
                TransparentUpgradeableProxy(
                    payable(address(registryCoordinator))
                ),
                address(registryCoordinatorImplementation),
                abi.encodeWithSelector(
                    regcoord.RegistryCoordinator.initialize.selector,
                    // we set churnApprover and ejector to communityMultisig because we don't need them
                    blocklessAVSCommunityMultisig,
                    blocklessAVSCommunityMultisig,
                    blocklessAVSCommunityMultisig,
                    blocklessAVSPauserReg,
                    0, // 0 initialPausedStatus means everything unpaused
                    quorumsOperatorSetParams,
                    quorumsMinimumStake,
                    quorumsStrategyParams
                )
            );
        }

        blocklessAVSImplementation = new BlocklessAVS(
            avsDirectory,
            registryCoordinator,
            stakeRegistry
        );
        // Third, upgrade the proxy contracts to use the correct implementation contracts and initialize them.
        blocklessAVSProxyAdmin.upgradeAndCall(
            TransparentUpgradeableProxy(
                payable(address(blocklessAVS))
            ),
            address(blocklessAVSImplementation),
            abi.encodeWithSelector(
                blocklessAVS.initialize.selector,
                blocklessAVSPauserReg,
                blocklessAVSCommunityMultisig,
                AGGREGATOR_ADDR
            )
        );

        // WRITE JSON DATA
        string memory parent_object = "parent object";

        string memory deployed_addresses = "addresses";
        // TODO: refactor
        vm.serializeAddress(
            deployed_addresses,
            "erc20Mock",
            address(erc20Mock)
        );
        // TODO: refactor
        vm.serializeAddress(
            deployed_addresses,
            "erc20MockStrategy",
            address(erc20MockStrategy)
        );
        vm.serializeAddress(
            deployed_addresses,
            "blocklessAVSServiceManager",
            address(blocklessAVS)
        );
        vm.serializeAddress(
            deployed_addresses,
            "blocklessAVSServiceManagerImplementation",
            address(blocklessAVSImplementation)
        );
        vm.serializeAddress(
            deployed_addresses,
            "registryCoordinator",
            address(registryCoordinator)
        );
        vm.serializeAddress(
            deployed_addresses,
            "registryCoordinatorImplementation",
            address(registryCoordinatorImplementation)
        );
        string memory deployed_addresses_output = vm.serializeAddress(
            deployed_addresses,
            "operatorStateRetriever",
            address(operatorStateRetriever)
        );

        // serialize all the data
        string memory finalJson = vm.serializeString(
            parent_object,
            deployed_addresses,
            deployed_addresses_output
        );

        writeOutput(finalJson, "blockless_avs_deployment_output");
    }
}
