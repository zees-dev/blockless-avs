package operator

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"github.com/pkg/errors"
)

// // Structs to unmarshal JSON data
// type Coin struct {
// 	ID     string `json:"id"`
// 	Symbol string `json:"symbol"`
// 	Name   string `json:"name"`
// }

type Price struct {
	USD float64 `json:"usd"`
}

// // Function to get the coin ID by symbol
// func getCoinID(symbol string) (string, error) {
// 	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/list")
// 	if err != nil {
// 		return "", errors.Wrap(err, "failed to get coin list")
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", errors.Wrap(err, "failed to read response body")
// 	}

// 	var coins []Coin
// 	if err := json.Unmarshal(body, &coins); err != nil {
// 		return "", errors.Wrap(err, "failed to unmarshal coin list")
// 	}

// 	for _, coin := range coins {
// 		if coin.Symbol == symbol {
// 			fmt.Println(coin)
// 			return coin.ID, nil
// 		}
// 	}

// 	return "", fmt.Errorf("symbol not found")
// }

// Function to get the price by coin ID
func getPriceByID(id string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", id)
	resp, err := http.Get(url)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get price")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read response body")
	}

	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal price")
	}

	price, ok := result[id]["usd"]
	if !ok {
		return 0, fmt.Errorf("price not found")
	}

	return price, nil
}

// Function to format the price to 6 decimal places and return it as uint
func formatPriceToSixDecimals(price float64) uint {
	return uint(math.Round(price * 1000000))
}
