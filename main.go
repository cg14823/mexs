package main

import (
	"encoding/csv"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"mexs/bots"
	"mexs/common"
	"mexs/exchange"
	"os"
	"path/filepath"
	"sort"
	"time"
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
// TODO: create a tool for schedule generation

func main() {
	// NOTE: proof of concept market experiment
	//nOfAgents := 30
	log.Debug("Starting Main")
	rand.Seed(time.Now().UnixNano())
	GAp := exchange.AuctionParameters{
		BidAskRatio:  0.5,
		KPricing:     0.5,
		MinIncrement: 1,
		MaxShift:     1,
		Dominance:    0,
	}

	marketInfo := common.MarketInfo{
		MaxPrice:     100.0,
		MinPrice:     1.0,
		MinIncrement: GAp.MinIncrement,
		MarketEnd:    10,
		TradingDays:  1,
	}

	sellersN := 30
	buyersN := 30
	// NOTE: Use the same for now
	sellerPrices := generateSteppedPrices(5.0, 1, 0, sellersN)
	buyerPrices := generateSteppedPrices(5.0, 1.0, 0, buyersN)
	sellerIDs := make([]int, sellersN)
	buyerIDs := make([]int, buyersN)

	log.Debug("SellerPrices:", sellerPrices)
	log.Debug("BuyerPrices:", buyerPrices)

	traders := make(map[int]bots.RobotTrader)
	for i := 0; i < buyersN; i++ {
		zip := &bots.ZIPTrader{}
		zip.InitRobotCore(i, "BUYER", marketInfo)
		zip.AddOrder(&bots.TraderOrder{
			LimitPrice: buyerPrices[i],
			Quantity:   1,
			Type:       "BID",
		})
		traders[zip.Info.TraderID] = zip
		buyerIDs[i] = zip.Info.TraderID
	}

	for i := 0; i < sellersN; i++ {
		zip := &bots.ZIPTrader{}
		zip.InitRobotCore(i+buyersN, "sellers", marketInfo)
		zip.AddOrder(&bots.TraderOrder{
			LimitPrice: sellerPrices[i],
			Quantity:   1,
			Type:       "ASK",
		})
		traders[zip.Info.TraderID] = zip
		sellerIDs[i] = zip.Info.TraderID
	}

	s := generateBasicAllocationSchedule(traders)

	ex := exchange.Exchange{}
	ex.Init(GAp, marketInfo, sellerIDs, buyerIDs)
	ex.SetTraders(traders)
	experimentID := uuid.New()
	ex.StartMarket(experimentID.String(), s)
	supplyAndDemandToCSV(sellerPrices, buyerPrices, experimentID.String(), "1")
}

// generateSteppedPrices creates limit prices
// @param min :- minimum value
// @param noise :- rand [-noise, ..., noise] added to values
// @param n is the number of prices to generate
func generateSteppedPrices(min, step float64, noise, n int) []float64 {
	prices := make([]float64, n)
	if noise != 0 {

		for i := 0; i < n; i++ {
			prices[i] = min + float64(i)*step + float64(rand.Intn(2*noise)-noise)
		}

		return prices
	}

	for i := 0; i < n; i++ {
		prices[i] = min + float64(i)*step
	}

	return prices
}

func generateBasicAllocationSchedule(traders map[int]bots.RobotTrader) exchange.AllocationSchedule {
	// This generates an allocation schedule in which all traders get the same order each training day

	s := exchange.AllocationSchedule{
		Schedule: make(map[int]map[int][]exchange.TID2RTO),
	}
	s.Schedule[-1] = make(map[int][]exchange.TID2RTO)
	s.Schedule[-1][0] = make([]exchange.TID2RTO, 0)

	for k, v := range traders {
		tido := exchange.TID2RTO{
			TraderID:  k,
			ExecOrder: v.GetExecutionOrder(),
		}
		s.Schedule[-1][0] = append(s.Schedule[-1][0], tido)
	}
	return s
}

func supplyAndDemandToCSV(sellers, buyers []float64, experimentID string, number string) {
	sort.Sort(float64arr(sellers))
	sort.Sort(sort.Reverse(float64arr(buyers)))

	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/LIMITPRICES_%s.csv", experimentID, number))
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}

	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("Limit prices CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Type 1 is ask, type 2 is bid
	writer.Write([]string{"NUMBER", "TYPE", "LIMIT_PRICE"})
	for _, v := range sellers {
		writer.Write([]string{
			number,
			"ASK",
			fmt.Sprintf("%.3f", v),
		})
	}

	for _, v := range buyers {
		writer.Write([]string{
			number,
			"BID",
			fmt.Sprintf("%.3f", v),
		})
	}

	log.Debug("Trades saved to file:", fileName)
}

type float64arr []float64

func (a float64arr) Len() int           { return len(a) }
func (a float64arr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a float64arr) Less(i, j int) bool { return a[i] < a[j] }
