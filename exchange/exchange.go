package exchange

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"mexs/bots"
	"mexs/common"
	"os"
	"path/filepath"
	"strconv"
	"time"
	fastRand "math/rand"
)

// AuctionParameters are the ones to be evolved by the GA
type AuctionParameters struct {
	// BidAskRatio is the bids/(bids +asks)
	BidAskRatio float64 `json:"BidAskRatio"`
	// k coefficient in k pricing rule pF = k *pB + (1-k)pA
	KPricing float64 `json:"KPricing"`
	// NOTE: For now use EE instead of MinIncrement
	// The minimum increment in the next bid
	// If it is 0 it means there is no shout/spread improvement
	// This is to implement some sort of NYSE shout improvement
	MinIncrement float64 `json:"MinIncrement"`
	// MaxShift is the maximum percentage a trader can move the current price
	MaxShift float64 `json:"MaxShift"`
	// Dominance defines how many traders have to trade before the
	// same trader is allowed to put in a bid/ask again 0 means no dominance
	Dominance int `json:"Dominance"`
	// Sliding Window size for EE shout improvement rule
	WindowSizeEE int `json:"WindowSizeEE"`
	// DeltaEE is the relaxing parameter in EE shout improvement rule
	DeltaEE float64 `json:"DeltaEE"`
	// OrderQueueing is the number of orders one trader can have queued
	// for now fixed to 1
	OrderQueuing int `json:"OrderQueuing,omitempty"`
}

// Used to define when
type AllocationSchedule struct {
	// The first key maps to a trading day
	// key -1 means all trading days
	// second key maps to trading step

	Schedule map[int]map[int]int
}

// Help structure to pair orders to Traders
type SandD struct {
	ID int
	SIDs []int
	BIDs []int
	Sps []AgentLimitPrices
	Bps []AgentLimitPrices
}
type SchedToPrices struct {
	SID int`json:"SID"`
	Day int `json:"Day, omitempty"`
	TimeStep int `json:"TimeStep, omitempty"`
	SLimitPrices []AgentLimitPrices `json:"SLimitPrices, omitempty"`
	BLimitPrices []AgentLimitPrices `json:"BLimitPrices, omitempty"`
}

type AgentLimitPrices struct {
	ID int `json:"ID"`
	Prices []float64 `json:"LimitPrice"`
	// Ignore for now
	Quantities []int64 `json:"Quantities, omitempty"`
}

/* Exchange defines the basic interfaces all exchanges have to follow
* <h3>Functions</>
*   - StartUp
*   -
 */
type Exchange struct {
	EID              string
	GAVector         AuctionParameters
	Info             common.MarketInfo
	orderBook        OrderBook
	agents           map[int]bots.RobotTrader
	AgentNum         int
	SellersIDs       []int
	BuyersIDs        []int
	bids             int
	asks             int
	trades           int
	tradeRecordPrice []float64
	SandDs map[int]SandD
	Alloc AllocationSchedule
	LogAll bool
}

func (ex *Exchange) Init(GAVector AuctionParameters, Info common.MarketInfo, sellers, buyers []int) {
	ex.GAVector = GAVector
	ex.Info = Info
	ex.orderBook = OrderBook{}
	ex.orderBook.Init()
	ex.agents = map[int]bots.RobotTrader{}
	ex.AgentNum = 0
	ex.SellersIDs = sellers
	ex.BuyersIDs = buyers
	ex.bids = 0
	ex.asks = 0
	ex.trades = 0
	ex.tradeRecordPrice = make([]float64, GAVector.WindowSizeEE)
}

func (ex *Exchange) SetTraders(traders map[int]bots.RobotTrader) {
	ex.agents = traders
	ex.AgentNum = len(traders)
}

func (ex *Exchange) ResetBidAskCount() {
	ex.bids = 0
	ex.asks = 0
}

func (ex *Exchange) PriceMatch(bid, ask *common.Order) *common.Trade {
	// Implement K pricing rule using GA parameter
	price := ex.GAVector.KPricing*bid.Price +
		(1-ex.GAVector.KPricing)*ask.Price

	return &common.Trade{
		TradeID:   ex.orderBook.GetNextTradeID(),
		Price:     price,
		BuyOrder:  bid,
		SellOrder: ask,
	}
}

func (ex *Exchange) MakeTrades(timeStep, d int) {
	// NOTE: This function is designed for the time step, single unit approach
	// for a multi unit asynchronous system this function should be called
	// every time an order is received and block until end
	// arriving orders should be put in a queued and processed in turn
	// The trade matching function should also be extended to deal with
	// multiple units
	ok, bid, ask, _ := ex.orderBook.FindPossibleTrade()
	if !ok {
		log.WithFields(log.Fields{
			"Time step": timeStep,
		}).Debug("No trade could be made")
		return
	}

	trade := ex.PriceMatch(bid, ask)
	trade.TimeStep = timeStep
	// NOTE: Should always be 1 for now it may be changed
	trade.Quantity = 1
	trade.Time = time.Now()

	// NOTE: This code is smelly, it assumes agents accept trade and can not refuse
	// once the order is posted for any reason
	err := ex.orderBook.RecordTrade(trade)
	if err != nil {
		log.WithFields(log.Fields{
			"Time step": timeStep,
		}).Error("Trade could not be made Error:", err)
		return
	}

	// add trade price to trade record to use with EE shout improvement rule
	ex.tradeRecordPrice[ex.trades%ex.GAVector.WindowSizeEE] = trade.Price
	ex.trades++

	//ex.agents[bid.TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, trade.TimeStep, trade.TradeID, trade.Price)
	//ex.agents[ask.TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, trade.TimeStep, trade.TradeID, trade.Price)
	// Traders should add there limit prices
	_, vl := ex.agents[bid.TraderID].TradeMade(trade)
	_, sl := ex.agents[ask.TraderID].TradeMade(trade)

	trade.BLimit = vl
	trade.SLimit = sl

	//ex.agents[bid.TraderID].LogBalance("../mexs/logs/"+ex.EID, d, trade)
	//ex.agents[ask.TraderID].LogBalance("../mexs/logs/"+ex.EID, d, trade)
	log.WithFields(log.Fields{
		"Time step": timeStep,
		"BuyerID":   bid.TraderID,
		"SellerID":  ask.TraderID,
		"Price":     trade.Price,
	}).Info("Trade made!")
}

func (ex *Exchange) GetTraderOrder(t, d int, eid string) (bool, *common.Order) {
	traderType := "NONE"
	for tries := 0; tries < 5; tries++ {
		traderType1, traderID := ex.EnforceBidToAskRatio(traderType, t)
		traderType = traderType1
		var agent bots.RobotTrader = ex.agents[traderID]
		order := agent.GetOrder(t)
		if order.OrderType != "BID" && order.OrderType != "ASK" {
			log.WithFields(log.Fields{
				"traderType":       traderType,
				"OrderType":        order.OrderType,
				"TraderID":         traderID,
				"Agent has orders": len(agent.GetExecutionOrder()),
			}).Debug("invalid order")
			continue
		}

		validOrder, reason := ex.OrderComplies(order, t)
		if validOrder {
			if order.OrderType == "BID" {
				ex.bids++
			} else {
				ex.asks++
			}
			log.Debugf("Bids ask %d:%d", ex.bids, ex.asks)
			if (ex.LogAll) {
				logOrderToCSV(order, d, eid, "TRUE", "N/A")
			}
			return true, order
		}

		log.WithFields(log.Fields{
			"traderType":       traderType,
			"OrderType":        order.OrderType,
			"order price":      order.Price,
			"TraderID":         traderID,
			"Agent has orders": len(agent.GetExecutionOrder()),
		}).Debug("Order did not comply")
		if (ex.LogAll) {
			logOrderToCSV(order, d, eid, "FALSE", reason)
		}
	}
	return false, &common.Order{}
}

func logOrderToCSV(order *common.Order, day int, experimentID, accepted, reason string) {
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/ALLORDERS.csv", experimentID))
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  day,
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}

	addHeader := true
	if _, err := os.Stat(fileName); err == nil {
		addHeader = false
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  day,
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("Trade CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "OrderType", "TID", "PRICE", "ACCEPTED", "REASON"})
	}

	writer.Write([]string{
		strconv.Itoa(day),
		strconv.Itoa(order.TimeStep),
		order.OrderType,
		strconv.Itoa(order.TraderID),
		fmt.Sprintf("%.5f", order.Price),
		accepted,
		reason,
	})
}

func (ex *Exchange) EnforceBidToAskRatio(traderType string, t int) (string, int) {
	if traderType == "NONE" {
		traderType2 := ex.nextBuyerOrSeller(t)
		return traderType2, ex.getRandomTrader(traderType2)
	}

	return traderType, ex.getRandomTrader(traderType)
}

// switch between enforce agent and random picker
func (ex *Exchange) RandomAgentPicker(traderType string, t int) (string, int) {
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(ex.AgentNum)))
	id := int(val.Int64())
	tType := "buyer"
	return tType, id
}

// Returns the id of the trader to be asked
func (ex *Exchange) nextBuyerOrSeller(t int) string {
	// case where no bids or ask made choose at random
	tOrders := ex.totalOrders()
	if tOrders == 0 {
		x := fastRand.Float64()
		if x < 0.5 {
			return "buyer"
		}
		return "seller"
	}
	// Choose at random if current BA is same as he one we want
	currentBa := ex.currentBA()
	if currentBa == ex.GAVector.BidAskRatio {
		x := fastRand.Float64()
		if x < 0.5 {
			return "buyer"
		}
		return "seller"
	}

	if currentBa < ex.GAVector.BidAskRatio {
		return "buyer"
	}

	return "seller"
}

func (ex *Exchange) totalOrders() int {
	return ex.bids + ex.asks
}

func (ex *Exchange) currentBA () float64 {
	return float64(ex.bids) / float64(ex.totalOrders())
}

func (ex *Exchange) getRandomTrader(traderType string) int {
	if traderType == "seller" {
		val, _ := rand.Int(rand.Reader, big.NewInt(int64(len(ex.SellersIDs))))
		return ex.SellersIDs[int(val.Int64())]
	} else if traderType == "buyer" {
		val, _ := rand.Int(rand.Reader, big.NewInt(int64(len(ex.BuyersIDs))))
		return ex.BuyersIDs[int(val.Int64())]
	}

	log.WithFields(log.Fields{
		"traderType": traderType,
	}).Error("Invalid trader type")
	// pick an Id at random
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(ex.AgentNum)))
	return int(val.Int64())
}

func (ex *Exchange) OrderComplies(order *common.Order, t int) (bool, string) {
	// It will check that the order follows the market rules

	//Between max and min values of the system
	if order.Price > ex.Info.MaxPrice || order.Price < ex.Info.MinPrice {
		return false, "Not invalid price range"
	}

	// Here we deal with the minimum increment rule NYSE style rule
	//valid := ex.ShoutImprovementRule(order)
	//if !valid {
	//	return valid, "Does not pass minimum increment"
	//}
	//EE shout improvemnt
	valid := ex.EEShoutImprovement(order)
	if !valid {
		return valid, "Does not pass minimum EE increment"
	}

	valid = ex.MaxShift(order)
	if !valid {
		return valid, "Does not pass max shift"
	}

	valid = ex.DominanceRule(order, t)
	if !valid {
		return valid, "Does not pass max shift"
	}
	return valid, "PASSES"
}

func (ex *Exchange) DominanceRule(order *common.Order, t int) bool {
	// Dominance rule is to ensure no trader can dominate by being the only one sending bids/asks
	if order.OrderType == "BID" {
		if lastO, ok := ex.orderBook.bidBook.Orders[order.TraderID]; ok {
			if lastO.TimeStep+ex.GAVector.Dominance > t {
				log.WithFields(log.Fields{
					"Dominance":         ex.GAVector.Dominance,
					"Last Quote in":     lastO.TimeStep,
					"Current Time step": t,
				}).Debug("Order rejected as it not passes Dominance rule")
				return false
			}
		}
	} else if order.OrderType == "ASK" {
		if lastO, ok := ex.orderBook.askBook.Orders[order.TraderID]; ok {
			if lastO.TimeStep+ex.GAVector.Dominance > t {
				log.WithFields(log.Fields{
					"Dominance":         ex.GAVector.Dominance,
					"Last Quote in":     lastO.TimeStep,
					"Current Time step": t,
				}).Debug("Order rejected as it not passes Dominance rule")
				return false
			}
		}
	}

	return true
}

func (ex *Exchange) EEShoutImprovement(order *common.Order) bool {
	// This rule https://www.researchgate.net/publication/221455475_Reducing_price_fluctuation_in_continuous_double_auctions_through_pricing_policy_and_shout_improvement
	// The rule keeps an estimate of the equilibrium price using an estimate of the equilibrium price Pe
	// Pe = (1/m) * sum_0_m(Pi)
	// where m is the size of the sliding window
	// then all bids above Pe - delta are accepted
	// and all asks bellow Pe + delta are accepted
	// Tunable parameters are m and delta
	if ex.trades < ex.GAVector.WindowSizeEE {
		// Not enough trades to estimate Pe so accept all bids and asks
		return true
	}

	pe := 1.0 / float64(ex.GAVector.WindowSizeEE)
	sum := 0.0
	for i := 0; i < ex.GAVector.WindowSizeEE; i++ {
		sum += ex.tradeRecordPrice[i]
	}
	pe = pe * sum

	if order.OrderType == "BID" && order.Price >= (pe-ex.GAVector.DeltaEE) {
		return true
	} else if order.OrderType == "ASK" && order.Price <= (pe+ex.GAVector.DeltaEE) {
		return true
	}

	return false
}

func (ex *Exchange) MaxShift(order *common.Order) bool {
	// This maximum shift rule defines the maximum amount any seller can shift the
	// Price, it is based on the last transaction
	if ex.orderBook.lastTrade.TradeID == -1 {
		return true
	}

	if ex.orderBook.lastTrade.Price*ex.GAVector.MaxShift >
		math.Abs(ex.orderBook.lastTrade.Price-order.Price) {
		return true
	}

	log.WithFields(log.Fields{
		"Max Shift":   ex.GAVector.MaxShift,
		"Last Trade":  ex.orderBook.lastTrade.Price,
		"Order Price": order.Price,
	}).Debug("Order rejected as it not passes max shift rule")
	return false
}

func (ex *Exchange) ShoutImprovementRule(order *common.Order) bool {
	if order.OrderType == "BID" {
		if ex.orderBook.bidBook.BestPrice != -1 {
			if ex.orderBook.bidBook.BestPrice+ex.GAVector.MinIncrement > order.Price {
				log.WithFields(log.Fields{
					"Best price":        ex.orderBook.bidBook.BestPrice,
					"Minimum Increment": ex.GAVector.MinIncrement,
					"Bid":               order.Price,
					"TID":               order.TraderID,
				}).Debug("Bid was rejected as it does not improve enough on best bid")
				return false
			}
		}
	} else {
		if ex.orderBook.askBook.BestPrice != -1 {
			if ex.orderBook.askBook.BestPrice-ex.GAVector.MinIncrement < order.Price {
				log.WithFields(log.Fields{
					"Best ask":          ex.orderBook.askBook.BestPrice,
					"Minimum Increment": ex.GAVector.MinIncrement,
					"Ask":               order.Price,
					"TID":               order.TraderID,
				}).Debug("Ask was rejected as it does not improve enough on best ask")
				return false
			}
		}
	}

	return true
}

func (ex *Exchange) UpdateAgents(timeStep, day int) {
	ex.orderBook.askBook.SetBestData()
	ex.orderBook.bidBook.SetBestData()
	bestAsk := ex.orderBook.askBook.BestPrice
	bestBid := ex.orderBook.bidBook.BestPrice

	marketUpdate := common.MarketUpdate{
		TimeStep:  timeStep,
		Day:       day,
		EID:       ex.EID,
		BestAsk:   bestAsk,
		BestBid:   bestBid,
		Bids:      ex.orderBook.bidBook.OrdersToList(),
		Asks:      ex.orderBook.askBook.OrdersToList(),
		Trades:    ex.orderBook.tradeRecord,
		LastTrade: ex.orderBook.lastTrade,
	}

	for _, agent := range ex.agents {
		agent.MarketUpdate(marketUpdate)
	}
}

func (ex *Exchange) StartMarket(experimentID string, s AllocationSchedule, sAndDs map[int]SandD) {
	ex.EID = experimentID
	ex.SandDs = sAndDs
	ex.Alloc = s

	log.WithFields(log.Fields{
		"Trading days":        ex.Info.TradingDays,
		"Training time steps": ex.Info.MarketEnd,
		"Num. Traders":        len(ex.agents),
		"ID":                  experimentID,
	}).Info("Market experiment started")

	err := os.MkdirAll("../mexs/logs/"+experimentID+"/", 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Log Folder for this experiment could not be made")
	}

	if (ex.LogAll) {
		ex.ScheduleToCSV(experimentID)
	}

	for d := 0; d < ex.Info.TradingDays; d++ {
		ex.orderBook.Reset()
		ex.ResetBidAskCount()
		log.Info("Trading day:", d)
		for t := 0; t < ex.Info.MarketEnd; t++ {
			log.Info("Time-step:", t)
			ex.RenewExecOrders(t, d)
			ok, order := ex.GetTraderOrder(t, d, experimentID)
			if ok {
				err := ex.orderBook.AddOrder(order)
				if err == nil {
					log.WithFields(log.Fields{
						"TID":   order.TraderID,
						"Type":  order.OrderType,
						"Price": order.Price,
					}).Info("Order received")
				} else {
					log.WithFields(log.Fields{
						"Time step": t,
						"error":     err.Error(),
					}).Error("Order could not be added")
				}
			} else {
				log.WithFields(log.Fields{
					"Time step": t,
				}).Info("No order was received this time step")
			}

			ex.MakeTrades(t, d)
			ex.orderBook.SetBest()
			ex.UpdateAgents(t, d)
		}

		log.WithFields(log.Fields{
			"Trades": len(ex.orderBook.tradeRecord),
			"Bids":   ex.bids,
			"Asks":   ex.asks,
		}).Info("Trading day ended")
		ex.orderBook.TradesToCSV(experimentID, d)
	}
	log.WithFields(log.Fields{
		"Trades":         ex.trades,
		"remaining Bids": len(ex.orderBook.bidBook.Orders),
		"remaining Asks": len(ex.orderBook.askBook.Orders),
		"EID":            experimentID,
	}).Info("Experiment ended")
}

// It renews Execution Orders based on a schedule
func (ex *Exchange) RenewExecOrders(t, d int) {
	// Check that there is a schedule relocation in day d at time t
	if _, ok := ex.Alloc.Schedule[d]; ok {
		if id, ok := ex.Alloc.Schedule[d][t]; ok {
			// Check that schedule with id:id exists
			if sandd, ok := ex.SandDs[id]; ok {
				// Set orders for sellers
				for _, lp := range sandd.Sps {
					orders := make([]*bots.TraderOrder, len(lp.Prices))
					for ix, p := range lp.Prices {
						// FIXME quantity hardcoded to 1
						order := &bots.TraderOrder{
							LimitPrice: p,
							Quantity: 1,
							Type:"ASK",
						}
						orders[ix] = order
					}
					ex.agents[lp.ID].SetOrders(orders)
				}
				// Set orders for buyers
				for _, lp := range sandd.Bps {
					orders := make([]*bots.TraderOrder, len(lp.Prices))
					for ix, p := range lp.Prices {
						// FIXME quantity hardcoded to 1
						order := &bots.TraderOrder{
							LimitPrice: p,
							Quantity: 1,
							Type:"BID",
						}
						orders[ix] = order
					}
					ex.agents[lp.ID].SetOrders(orders)
				}
				log.Debug("Traders Replentish")
			}
		}
	}
}

func (ex *Exchange) ScheduleToCSV(eid string) {
	// Schedules csv works in a sort of relational table way. there is the schedule csv that links
	// S&D ids to time step and trading dat
	//Create schedule.csv
	fileName1, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/schedule.csv", eid))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err.Error(),
		}).Error("Error creating schedule.scv")
		return
	}

	file1, err := os.OpenFile(fileName1, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file1.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err.Error(),
		}).Error("Sched CSV file could not be made")
		return
	}

	writer1 := csv.NewWriter(file1)
	defer writer1.Flush()
	writer1.Write([]string{"TradingDay", "TimeStep", "ScheduleID"})
	for d, _ := range ex.Alloc.Schedule {
		for t, id := range ex.Alloc.Schedule[d] {
			writer1.Write([]string{
				strconv.Itoa(d),
				strconv.Itoa(t),
				strconv.Itoa(id),
			})
		}
	}

	// Write limit prices for each schedule in csv
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/LimitPrices.csv", eid))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Error creating schedule.scv")
		return
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("CSV file could not be made: ", fileName)
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"ID", "TID", "TYPE", "LIMIT"})

	for id, sandd := range ex.SandDs {
		for _, slp := range sandd.Sps {
			for _, ps := range slp.Prices {
				writer.Write([]string{
					strconv.Itoa(id),
					strconv.Itoa(slp.ID),
					"ASK",
					fmt.Sprintf("%.2f", ps),
				})
			}
		}

		for _, blp := range sandd.Bps {
			for _, ps := range blp.Prices {
				writer.Write([]string{
					strconv.Itoa(id),
					strconv.Itoa(blp.ID),
					"BID",
					fmt.Sprintf("%.2f", ps),
				})
			}
		}
	}
}
