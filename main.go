package main

import (
	"mexs/bots"
	"mexs/common"
	"mexs/exchange"
	"math/rand"
	"time"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}


// TODO: create a cli interfaces to setup experiment with out having to go
// through the code
// TODO: create tools to automatically create limit prices, orders and
// instantiate traders

func main() {
	// NOTE: proof of concept market experiment
	//nOfAgents := 30
	log.Debug("Starting Main")
	rand.Seed(time.Now().UnixNano())
	GAp := exchange.AuctionParameters{
		BidAskRatio: 0.5,
		KPricing: 0.5,
		MinIncrement: 1,
		MaxShift: 0.2,
		Dominance: 0,
	}

	marketInfo := common.MarketInfo{
		MaxPrice: float32(100),
		MinPrice: float32(1),
		MarketEnd: 100,
		TradingDays: 1,
	}

	// NOTE: Use the same for now
	sellerPrices := generateSteppedPrices(1, float32(5), 0, 15)

	traders := make(map[int]bots.RobotTrader)
	for i :=0; i < 15; i ++ {
		zic := &bots.ZICTrader{}
		zic.InitRobotCore(i, "ZIC", "BUYER", marketInfo)
		zic.AddOrder(&bots.TraderOrder{
			LimitPrice: sellerPrices[i],
			Quantity: 1,
			Type: "BID",
		})
		traders[i] = zic
	}

	for i := 0; i < 15; i++ {
		zic := &bots.ZICTrader{}
		zic.InitRobotCore(i+15, "ZIC", "sellers", marketInfo)
		zic.AddOrder(&bots.TraderOrder{
			LimitPrice: sellerPrices[i],
			Quantity: 1,
			Type: "ASK",
		})
		traders[i+15] = zic
	}

	ex := exchange.Exchange{}
	ex.Init(GAp, marketInfo)
	ex.SetTraders(traders)
	ex.StartExperiment()
}
// generateSteppedPrices creates limit prices
// @param min :- minimum value
// @param noise :- rand [-noise, ..., noise] added to values
// @param n is the number of prices to generate
func generateSteppedPrices (min, step float32, noise, n int) []float32{
	prices := make([]float32, n)
	if noise != 0 {

		for i := 0; i < n; i++ {
			prices[i] = min + step + float32(rand.Intn(2*noise)-noise)
		}

		return prices
	}

	for i := 0; i < n; i++ {
		prices[i] = min + step
	}

	return prices
}
