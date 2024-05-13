// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.12;

import {BytesLib} from "@eigenlayer/contracts/libraries/BytesLib.sol";
import {ServiceManagerBase} from "@eigenlayer-middleware/src/ServiceManagerBase.sol";
import {BLSSignatureChecker, IRegistryCoordinator} from "@eigenlayer-middleware/src/BLSSignatureChecker.sol";
import {IBLSSignatureChecker} from "@eigenlayer-middleware/src/interfaces/IBLSSignatureChecker.sol";
import {OperatorStateRetriever} from "@eigenlayer-middleware/src/OperatorStateRetriever.sol";
import {IStakeRegistry} from "@eigenlayer-middleware/src/interfaces/IStakeRegistry.sol";
import {Pausable} from "@eigenlayer/contracts/permissions/Pausable.sol";
import {IAVSDirectory} from "@eigenlayer/contracts/interfaces/IAVSDirectory.sol";
import {IPauserRegistry} from "@eigenlayer/contracts/interfaces/IPauserRegistry.sol";

interface IBlocklessAVS {
    /*//////////////////////////////////////////////////////////////
                                CONSTANT
    //////////////////////////////////////////////////////////////*/

    function BLOCK_STALE_MEASURE() external pure returns (uint32);

    /*//////////////////////////////////////////////////////////////
                                STRUCTS
    //////////////////////////////////////////////////////////////*/

    /// @notice Price data for a symbol; price is in 6dp.
    /// @dev This is signed by operators to create a verifiable digest.
    struct Price {
        string symbol;
        uint256 price;
        uint32 timestamp;
    }

    /// @notice Oracle request to get price of a symbol.
    struct OracleRequest {
        string symbol;
        uint32 referenceBlockNumber;
        // task submitter decides on the criteria for a task to be completed
        // note that this does not mean the task was "correctly" answered (i.e. the number was squared correctly)
        //      this is for the challenge logic to verify
        // task is completed (and contract will accept its TaskResponse) when each quorumNumbers specified here
        // are signed by at least quorumThresholdPercentage of the operators
        // note that we set the quorumThresholdPercentage to be the same for all quorumNumbers, but this could be changed
        bytes quorumNumbers;
        uint8 quorumThresholdPercentage;
    }

    /// @notice Metadata for the oracle price response.
    struct OraclePriceResponseMetadata {
        uint32 blockNumber;
        bytes32 hashOfNonSigners;
    }

    // functions
    function aggregator() external view returns (address);
    function price(string calldata symbol) external view returns (Price memory);

    function updateOraclePrice(OracleRequest calldata oracleRequest, Price calldata priceResponse, IBLSSignatureChecker.NonSignerStakesAndSignature calldata nonSignerStakesAndSignature) external;
    function freezeOperator(address _operatorAddr) external;
    function setAggregator(address _newAggregator) external;
}

/**
 * @title Primary entrypoint for procuring services from Blockless.
 * @author Blockless.
 */
contract BlocklessAVS is IBlocklessAVS, ServiceManagerBase, BLSSignatureChecker, Pausable {
    using BytesLib for bytes;

    /*//////////////////////////////////////////////////////////////
                                CONSTANT
    //////////////////////////////////////////////////////////////*/

    /**
     * @notice The maximum amount of blocks in the past that the service will consider stake amounts to still be 'valid'.
     * @dev To clarify edge cases, the middleware can look `BLOCK_STALE_MEASURE` blocks into the past, i.e. it may trust stakes from the interval
     * [block.number - BLOCK_STALE_MEASURE, block.number] (specifically, *inclusive* of the block that is `BLOCK_STALE_MEASURE` before the current one)
     * @dev BLOCK_STALE_MEASURE should be greater than the number of blocks till finalization, but not too much greater, as it is the amount of
     * time that nodes can be active after they have deregistered. The larger it is, the farther back stakes can be used, but the longer operators
     * have to serve after they've deregistered.
     */
    uint32 public constant BLOCK_STALE_MEASURE = 150;

    uint256 internal constant _THRESHOLD_DENOMINATOR = 100;

    /*//////////////////////////////////////////////////////////////
                                STORAGE
    //////////////////////////////////////////////////////////////*/

    /// @notice The address of the aggregator contract.
    address public aggregator;

    /// @notice Mapping from symbol to latest Price data.
    mapping(string => Price) public prices;

    /*//////////////////////////////////////////////////////////////
                                MODIFIERS
    //////////////////////////////////////////////////////////////*/

    modifier onlyAggregator() {
        require(msg.sender == aggregator, "Aggregator must be the caller");
        _;
    }

    /*//////////////////////////////////////////////////////////////
                                EVENTS
    //////////////////////////////////////////////////////////////*/

    event OracleUpdate(Price priceResponse, OraclePriceResponseMetadata oraclePriceResponseMetadata);

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
    {
        _disableInitializers();
    }

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

    function price(string calldata symbol) external view returns (Price memory) {
        return prices[symbol];
    }

    /*//////////////////////////////////////////////////////////////
                            RESTRICTED FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    function updateOraclePrice(
        OracleRequest calldata oracleRequest,
        Price calldata priceResponse,
        NonSignerStakesAndSignature calldata nonSignerStakesAndSignature
    ) external onlyAggregator {
        require(oracleRequest.referenceBlockNumber <= uint32(block.number), "BlocklessAVS.updateOraclePrice: specified referenceBlockNumber is in future");
        require(
            (oracleRequest.referenceBlockNumber + BLOCK_STALE_MEASURE) >= uint32(block.number),
            "BlocklessAVS.updateOraclePrice: specified referenceBlockNumber is too far in past"
        );

        /* CHECKING SIGNATURES & WHETHER THRESHOLD IS MET OR NOT */
        // calculate message which operators signed
        bytes32 message = keccak256(abi.encode(priceResponse));

        // check the signature
        (
            QuorumStakeTotals memory quorumStakeTotals,
            bytes32 hashOfNonSigners
        ) = checkSignatures(
            message, 
            oracleRequest.quorumNumbers, // use list of uint8s instead of uint256 bitmap to not iterate 256 times
            oracleRequest.referenceBlockNumber, 
            nonSignerStakesAndSignature
        );

        // check that signatories own at least a threshold percentage of each quourm
        for (uint256 i = 0; i < oracleRequest.quorumNumbers.length; i++) {
            // we don't check that the quorumThresholdPercentages are not >100 because a greater value would trivially fail the check, implying
            // signed stake > total stake
            require(
                quorumStakeTotals.signedStakeForQuorum[i] * _THRESHOLD_DENOMINATOR >=
                quorumStakeTotals.totalStakeForQuorum[i] * uint8(oracleRequest.quorumThresholdPercentage),
                "BlocklessAVS.updateOraclePrice: signatories do not own at least threshold percentage of a quorum"
            );
        }

        // Validate oracle price
        validateOraclePrice(priceResponse);
        
        // Update latest price for symbol
        prices[priceResponse.symbol] = priceResponse;

        // Emit the new price event
        OraclePriceResponseMetadata memory oraclePriceResponseMetadata = OraclePriceResponseMetadata(
            uint32(block.number),
            hashOfNonSigners
        );
        emit OracleUpdate(priceResponse, oraclePriceResponseMetadata);
    }

    // TODO: look into introducing this function
    /// @notice Called in the event of challenge resolution, in order to forward a call to the Slasher, which 'freezes' the `operator`.
    /// @dev The Slasher contract is under active development and its interface expected to change.
    ///      We recommend writing slashing logic without integrating with the Slasher at this point in time.
    function freezeOperator(address operatorAddr) external onlyAggregator {
        // slasher.freezeOperator(operatorAddr);
    }

    /*//////////////////////////////////////////////////////////////
                            INTERNAL FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    /// @dev only the aggregator can call this function.
    /// @param price The new price.
    function validateOraclePrice(Price calldata price) internal {
        // Check that the price and timestamp are valid
        require(price.price > 0, "BlocklessAVS.updatePrice: price must be positive");

        require(price.timestamp <= block.timestamp, "BlocklessAVS.updatePrice: timestamp cannot be in the future");

        // Ensure the timestamp is within the valid range
        uint256 timeWindow = 1 minutes; // or any desired time window
        require(price.timestamp >= block.timestamp - timeWindow, "BlocklessAVS.updatePrice: timestamp outside valid range");

        // Check that the price is different from the previous price within the time window
        Price storage prevPrice = prices[price.symbol];
        require(
            prevPrice.timestamp < block.timestamp - timeWindow || prevPrice.price != price.price,
            "BlocklessAVS.updatePrice: price already updated within time window"
        );
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
     * - BlocklessAVS: 1 (aggregator)
     * - OwnableUpgradeable: 1
     * - BLSSignatureChecker: 1
     * - Pausable: 2
     */
    uint256[45] private __GAP;
}
