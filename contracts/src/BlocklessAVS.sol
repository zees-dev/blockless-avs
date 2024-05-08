// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.9;

import {BytesLib} from "@eigenlayer/contracts/libraries/BytesLib.sol";
import {ServiceManagerBase} from "@eigenlayer-middleware/src/ServiceManagerBase.sol";
import {BLSSignatureChecker, IRegistryCoordinator} from "@eigenlayer-middleware/src/BLSSignatureChecker.sol";
import {OperatorStateRetriever} from "@eigenlayer-middleware/src/OperatorStateRetriever.sol";
import {IStakeRegistry} from "@eigenlayer-middleware/src/interfaces/IStakeRegistry.sol";
import {Pausable} from "@eigenlayer/contracts/permissions/Pausable.sol";
import {IAVSDirectory} from "@eigenlayer/contracts/interfaces/IAVSDirectory.sol";
import {IPauserRegistry} from "@eigenlayer/contracts/interfaces/IPauserRegistry.sol";

/**
 * @title Primary entrypoint for procuring services from Blockless.
 * @author Blockless.
 */
contract BlocklessAVS is ServiceManagerBase, BLSSignatureChecker, OperatorStateRetriever, Pausable {
    using BytesLib for bytes;

    /*//////////////////////////////////////////////////////////////
                                CONSTANT
    //////////////////////////////////////////////////////////////*/

    uint32 public constant TASK_CHALLENGE_WINDOW_BLOCK = 100;

    /*//////////////////////////////////////////////////////////////
                                STORAGE
    //////////////////////////////////////////////////////////////*/

    /// @notice The address of the aggregator contract.
    address public aggregator;

    /// @notice Mapping from symbol to Price.
    mapping(string => Price) public prices;

    /*//////////////////////////////////////////////////////////////
                                MODIFIERS
    //////////////////////////////////////////////////////////////*/

    modifier onlyAggregator() {
        require(msg.sender == aggregator, "Aggregator must be the caller");
        _;
    }

    /*//////////////////////////////////////////////////////////////
                                STRUCTS
    //////////////////////////////////////////////////////////////*/

    struct Price {
        uint256 price;
        uint256 timestamp;
    }

    /*//////////////////////////////////////////////////////////////
                                EVENTS
    //////////////////////////////////////////////////////////////*/

    event OracleUpdate(
        string indexed symbol,
        uint256 answer,
        uint256 timestamp,
        uint256 answeredAt,
        uint80 answeredInRound
    );

    event AggregatorUpdated(address indexed previousAggregator, address indexed newAggregator);

    /*//////////////////////////////////////////////////////////////
                                CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    constructor(
        IAVSDirectory _avsDirectory,
        IRegistryCoordinator _registryCoordinator,
        IStakeRegistry _stakeRegistry
    )
    ServiceManagerBase(_avsDirectory, _registryCoordinator, _stakeRegistry)
    BLSSignatureChecker(_registryCoordinator) // TODO: validate whether `staleStakesForbidden` should be set to `true`
    {}

    function initialize(
        IPauserRegistry _pauserRegistry,
        address _initialOwner,
        address _aggregator
    ) public initializer {
        _transferOwnership(_initialOwner);
        _initializePauser(_pauserRegistry, UNPAUSE_ALL);
        aggregator = _aggregator;
        emit AggregatorUpdated(address(0), _aggregator);
    }

    /*//////////////////////////////////////////////////////////////
                            PUBLIC FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @notice Get the latest price data for a symbol.
    function latestData(string memory symbol) external view returns (uint256, uint256, uint256, uint80) {
        Price storage price = prices[symbol];
        return (
            price.price,
            price.timestamp,
            block.timestamp,
            uint80(block.number)
        );
    }

    /*//////////////////////////////////////////////////////////////
                            RESTRICTED FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @notice Update the price for an asset.
    /// @dev only the aggregator can call this function.
    /// @param symbol The symbol of the asset.
    /// @param price The new price.
    /// @param timestamp The timestamp of the price.
    function updatePrice(string memory symbol, uint256 price, uint256 timestamp) external onlyAggregator {
        // Check that the price and timestamp are valid
        require(price > 0, "PriceOracle: price must be positive");
        require(
            timestamp <= block.timestamp,
            "PriceOracle: timestamp cannot be in the future"
        );

        // Ensure the timestamp is within the valid range
        uint256 timeWindow = 1 minutes; // or any desired time window
        require(
            timestamp >= block.timestamp - timeWindow,
            "PriceOracle: timestamp outside valid range"
        );

        // Check that the price is different from the previous price within the time window
        Price storage prevPrice = prices[symbol];
        require(
            prevPrice.timestamp < block.timestamp - timeWindow ||
                prevPrice.price != price,
            "PriceOracle: price already updated within time window"
        );

        // Update the price for the specified asset
        prices[symbol] = Price(price, timestamp);

        // Emit the new price event
        emit OracleUpdate(
            symbol,
            price,
            timestamp,
            block.timestamp,
            uint80(block.number)
        );
    }

    /// @notice Called in the event of challenge resolution, in order to forward a call to the Slasher, which 'freezes' the `operator`.
    /// @dev The Slasher contract is under active development and its interface expected to change.
    ///      We recommend writing slashing logic without integrating with the Slasher at this point in time.
    function freezeOperator(address operatorAddr) external onlyAggregator {
        // slasher.freezeOperator(operatorAddr);
    }

    /*//////////////////////////////////////////////////////////////
                            ADMIN FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @notice Set the address of the aggregator contract.
    function setAggregator(address _newAggregator) external onlyOwner {
        address oldAggregator = aggregator;
        aggregator = _newAggregator;
        emit AggregatorUpdated(oldAggregator, _newAggregator);
    }

    // slither-disable-next-line shadowing-state
    /**
     * @notice storage gap for upgradeability
     * @dev Storage gap for upgradeability.
     * Slots used:
     * - OwnableUpgradeable: 1
     * - BLSSignatureChecker: 1
     * - Pausable: 2
     */
    uint256[46] private __GAP;
}
