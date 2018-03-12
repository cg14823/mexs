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
)

// AuctionParameters are the ones to be evolved by the GA
type AuctionParameters struct {
	// BidAskRatio is the proportion buyers to sellers
	BidAskRatio float64 `json:"BidAskRatio"`
	// k coefficient in k pricing rule pF = k *pB + (1-k)pA
	KPricing float64 `json:"KPricing"`
	// The minimum increment in the next bid
	// If it is 0 it means there is no shout/spread improvement
	MinIncrement float64 `json:"MinIncrement"`
	// MaxShift is the maximum percentage a trader can move the current price
	MaxShift float64 `json:"MaxShift"`
	// Dominance defines how many traders have to trade before the
	// same trader is allowed to put in a bid/ask again 0 means no dominance
	Dominance int `json:"Dominance"`
	// OrderQueueing is the number of orders one trader can have queued
	// for now fixed to 1
	OrderQueuing int `json:"OrderQueuing,omitempty"`
}

// Used to define when
type AllocationSchedule struct {
	// The first key maps to a trading day
	// key -1 means all trading days
	// second key maps to trading step

	Schedule map[int]map[int][]TID2RTO
}

// Help structure to pair orders to Traders
type TID2RTO struct {
	TraderID  int
	ExecOrder []*bots.TraderOrder
}

/* Exchange defines the basic interfaces all exchanges have to follow
* <h3>Functions</>
*   - StartUp
*   -
 */
type Exchange struct {
	EID        string
	GAVector   AuctionParameters
	Info       common.MarketInfo
	orderBook  OrderBook
	agents     map[int]bots.RobotTrader
	AgentNum   int
	SellersIDs []int
	BuyersIDs  []int
	bids       int
	asks       int
	trades     int
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
	ex.trades++
	if err != nil {
		log.WithFields(log.Fields{
			"Time step": timeStep,
		}).Error("Trade could not be made Error:", err)
		return
	}

	ex.agents[bid.TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, trade.TimeStep, trade.TradeID, trade.Price)
	ex.agents[ask.TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, trade.TimeStep, trade.TradeID, trade.Price)
	ex.agents[bid.TraderID].TradeMade(trade)
	ex.agents[ask.TraderID].TradeMade(trade)
	ex.agents[bid.TraderID].LogBalance("../mexs/logs/"+ex.EID, d, trade)
	ex.agents[ask.TraderID].LogBalance("../mexs/logs/"+ex.EID, d, trade)
	log.WithFields(log.Fields{
		"Time step": timeStep,
		"BuyerID":   bid.TraderID,
		"SellerID":  ask.TraderID,
		"Price":     trade.Price,
	}).Info("Trade made!")
}

func (ex *Exchange) GetTraderOrder(t, d int, eid string) (bool, *common.Order) {
	traderType := "NONE"
	for tries := 0; tries < 10; tries++ {
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
			logOrderToCSV(order, d, eid, "TRUE", "N/A")
			return true, order
		}

		log.WithFields(log.Fields{
			"traderType":       traderType,
			"OrderType":        order.OrderType,
			"order price":      order.Price,
			"TraderID":         traderID,
			"Agent has orders": len(agent.GetExecutionOrder()),
		}).Debug("Order did not comply")
		logOrderToCSV(order, d, eid, "FALSE", reason)
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
	// First we deal with the edge cases of when the bid:ask is either 0 or INf
	if math.IsInf(ex.GAVector.BidAskRatio, 0) {
		// Only bids
		if numBuyers := len(ex.BuyersIDs); numBuyers > 0 {
			return "buyer"
		}
		log.WithFields(log.Fields{
			"Bid:Ask":  ex.GAVector.BidAskRatio,
			"#Sellers": len(ex.SellersIDs),
			"#Buyers":  len(ex.BuyersIDs),
			"Function": "NextBuyerOrSeller",
		}).Error("Error Next is buyer but the are no buyers")
		panic("No buyers")
	}

	if ex.GAVector.BidAskRatio == 0 {
		// Only asks
		if numSellers := len(ex.SellersIDs); numSellers > 0 {
			return "seller"
		}

		log.WithFields(log.Fields{
			"Bid:Ask":  ex.GAVector.BidAskRatio,
			"#Sellers": len(ex.SellersIDs),
			"#Buyers":  len(ex.BuyersIDs),
			"Function": "enforceBidToAskRatio",
		}).Error("Error selecting random seller")
		panic("No Sellers")
	}

	currentRatio := float64(ex.bids) / float64(ex.asks)
	log.WithFields(log.Fields{
		"Bid:Ask":  ex.GAVector.BidAskRatio,
		"#Sellers": len(ex.SellersIDs),
		"#Buyers":  len(ex.BuyersIDs),
		"Function": "enforceBidToAskRatio",
		"Timestep": t,
	}).Debug("current bid ask before this time step :", currentRatio)

	if currentRatio > ex.GAVector.BidAskRatio {
		// this means that we need to lower our ratio thus select aks
		return "seller"
	} else if currentRatio < ex.GAVector.BidAskRatio {
		return "buyer"
	}

	// If ratio correct just pick one at random
	if val, _ := rand.Int(rand.Reader, big.NewInt(10)); val.Int64() >= 5 {
		return "seller"
	}

	return "buyer"
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
	// bid/ask ratio is controlled by the number of sellers versus the
	// number of buyers assuming that the next trader is chosen perfectly at random
	// then bid/ask ratio should tend to be equal to buyer/seller ratio

	//Between max and min values of the system
	if order.Price > ex.Info.MaxPrice || order.Price < ex.Info.MinPrice {
		return false, "Not invalid price range"
	}

	valid := true
	// Here we deal with the minimum increment rule
	//valid := ex.MinimumIncrementRule(order)
	//if !valid {
	//	return valid, "Does not pass minimum increment"
	//}

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
	}

	return true
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

func (ex *Exchange) MinimumIncrementRule(order *common.Order) bool {
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

func (ex *Exchange) StartMarket(experimentID string, s AllocationSchedule) {
	ex.EID = experimentID
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

	for d := 0; d < ex.Info.TradingDays; d++ {
		// NOTE: clear orderbook at start of each day
		ex.orderBook.Reset()
		ex.ResetBidAskCount()
		log.Info("Trading day:", d)
		for t := 0; t < ex.Info.MarketEnd; t++ {
			log.Info("Time-step:", t)
			ex.RenewExecOrders(t, d, s)

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
			ex.UpdateAgents(t, d)
		}

		log.WithFields(log.Fields{
			"Trades": len(ex.orderBook.tradeRecord),
			"Bids":   ex.bids,
			"Asks":   ex.asks,
		}).Info("Trading day ended")
		// TODO: store order books in database at the end of each day
		ex.orderBook.TradesToCSV(experimentID, d)
	}
	// TODO: Persistent save of data
	log.WithFields(log.Fields{
		"Trades":         ex.trades,
		"remaining Bids": len(ex.orderBook.bidBook.Orders),
		"remaining Asks": len(ex.orderBook.askBook.Orders),
		"EID":            experimentID,
	}).Info("Experiment ended")
}

// It renews Execution Orders based on a schedule
func (ex *Exchange) RenewExecOrders(t, d int, s AllocationSchedule) {
	if _, ok := s.Schedule[d]; ok {
		if _, ok := s.Schedule[d][t]; ok {
			orders := s.Schedule[d][t]
			for i := 0; i < len(orders); i++ {
				ex.agents[orders[i].TraderID].SetOrders(orders[i].ExecOrder)
				ex.agents[orders[i].TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, t, -1, -1.0)

			}
		}
	}

	if _, ok := s.Schedule[-1]; ok {
		if _, ok := s.Schedule[-1][t]; ok {
			orders := s.Schedule[-1][t]
			for i := 0; i < len(orders); i++ {
				ex.agents[orders[i].TraderID].SetOrders(orders[i].ExecOrder)
				ex.agents[orders[i].TraderID].LogOrder("../mexs/logs/"+ex.EID+"/ExecOrders.csv", d, t, -1, -1.0)
			}
		}
	}

	log.Debug("Renewed agents orders")
}
