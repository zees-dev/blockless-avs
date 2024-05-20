package core

import (
	"math/big"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
	"github.com/ethereum/go-ethereum/accounts/abi"
	csavs "github.com/zees-dev/blockless-avs/contracts/bindings/BlocklessAVS"
	"golang.org/x/crypto/sha3"
)

// this hardcodes abi.encode() for csavs.IBlocklessAVSPrice
// unclear why abigen doesn't provide this out of the box...
func AbiEncodePriceResponse(h *csavs.IBlocklessAVSPrice) ([]byte, error) {

	// The order here has to match the field ordering of csavs.IBlocklessAVSPrice
	priceType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{
			Name: "symbol",
			Type: "string",
		},
		{
			Name: "price",
			Type: "uint256",
		},
		{
			Name: "timestamp",
			Type: "uint32",
		},
	})
	if err != nil {
		return nil, err
	}
	arguments := abi.Arguments{
		{
			Type: priceType,
		},
	}

	bytes, err := arguments.Pack(h)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// GetPriceDigest returns the hash of the Price, which is what operators sign over
func GetPriceDigest(p *csavs.IBlocklessAVSPrice) ([32]byte, error) {

	encodePriceByte, err := AbiEncodePriceResponse(p)
	if err != nil {
		return [32]byte{}, err
	}

	var priceDigest [32]byte
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(encodePriceByte)
	copy(priceDigest[:], hasher.Sum(nil)[:32])

	return priceDigest, nil
}

// BINDING UTILS - conversion from contract structs to golang structs

// BN254.sol is a library, so bindings for G1 Points and G2 Points are only generated
// in every contract that imports that library. Thus the output here will need to be
// type casted if G1Point is needed to interface with another contract (eg: BLSPublicKeyCompendium.sol)
func ConvertToBN254G1Point(input *bls.G1Point) csavs.BN254G1Point {
	output := csavs.BN254G1Point{
		X: input.X.BigInt(big.NewInt(0)),
		Y: input.Y.BigInt(big.NewInt(0)),
	}
	return output
}

func ConvertToBN254G2Point(input *bls.G2Point) csavs.BN254G2Point {
	output := csavs.BN254G2Point{
		X: [2]*big.Int{input.X.A1.BigInt(big.NewInt(0)), input.X.A0.BigInt(big.NewInt(0))},
		Y: [2]*big.Int{input.Y.A1.BigInt(big.NewInt(0)), input.Y.A0.BigInt(big.NewInt(0))},
	}
	return output
}
