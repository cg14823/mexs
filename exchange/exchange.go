package exchange

import (
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"math"
	"math/big"
	"mexs/bots"
	"mexs/common"
	"time"
)

// AuctionParameters are the ones to be evolved by the GA
type AuctionParameters struct {
	// BidAskRatio is the proportion buyers to sellers
	BidAskRatio float64
	// k coefficient in k pricing rule pF = k *pB + (1-k)pA
	KPricing float64
	// The minimum increment in the next bid
	// If it is 0 it means there is no shout/spread improvement
	MinIncrement float64
	// MaxShift is the maximum percentage a trader can move the current price
	MaxShift float64
	// Dominance defines how many traders have to trade before the
	// same trader is allowed to put in a bid/ask again 0 means no dominance
	Dominance int
	// OrderQueueing is the number of orders one trader can have queued
	// for now fixed to 1
	OrderQueuing int
}

/* Exchange defines the basic interfaces all exchanges have to follow
* <h3>Functions</>
*   - StartUp
*   -
 */
type Exchange struct {
	GAVector  AuctionParameters
	Info      common.MarketInfo
	orderBook OrderBook
	agents    map[int]bots.RobotTrader
	AgentNum  int
}

func (ex *Exchange) Init(GAVector AuctionParameters, Info common.MarketInfo) {
	ex.GAVector = GAVector
	ex.Info = Info
	ex.orderBook = OrderBook{}
	ex.orderBook.Init()
	ex.agents = map[int]bots.RobotTrader{}
	ex.AgentNum = 0
}

func (ex *Exchange) SetTraders(traders map[int]bots.RobotTrader) {
	ex.agents = traders
	ex.AgentNum = len(traders)
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

func (ex *Exchange) MakeTrades(timeStep int) {
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

	ex.agents[bid.TraderID].TradeMade(trade)
	ex.agents[ask.TraderID].TradeMade(trade)
	log.WithFields(log.Fields{
		"Time step": timeStep,
		"BuyerID":   bid.TraderID,
		"SellerID":  ask.TraderID,
		"Price":     trade.Price,
	}).Info("Trade made!")
}

func (ex *Exchange) GetTraderOrder(t int) (bool, *common.Order) {
	for tries := 0; tries < 5; tries++ {
		traderIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(ex.AgentNum)))
		var agent bots.RobotTrader = ex.agents[int(traderIndex.Int64())]

		order := agent.GetOrder(t)
		if order.OrderType != "BID" && order.OrderType != "ASK" {
			continue
		}

		validOrder := ex.OrderComplies(order, t)
		if validOrder {
			return true, order
		}
	}
	return false, &common.Order{}
}

func (ex *Exchange) OrderComplies(order *common.Order, t int) bool {
	// It will check that the order follows the market rules
	// bid/ask ratio is controlled by the number of sellers versus the
	// number of buyers assuming that the next trader is chosen perfectly at random
	// then bid/ask ratio should tend to be equal to buyer/seller ratio

	//KPricing is implemented in the PriceMatch function

	// Here we deal with the minimum increment rule
	valid := ex.MinimumIncrementRule(order)
	if !valid {
		return valid
	}

	valid = ex.MaxShift(order)
	if !valid {
		return valid
	}

	valid = ex.DominanceRule(order, t)
	return valid
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
				}).Debug("Bid was rejected as it does not improve enough on best bid")
				return false
			}
		}
	} else {
		if ex.orderBook.askBook.BestPrice != -1 {
			if ex.orderBook.askBook.BestPrice-ex.GAVector.MinIncrement < order.Price {
				log.WithFields(log.Fields{
					"Best price":        ex.orderBook.askBook.BestPrice,
					"Minimum Increment": ex.GAVector.MinIncrement,
					"Ask":               order.Price,
				}).Debug("Ask was rejected as it does not improve enough on best bid")
				return false
			}
		}
	}

	return true
}

func (ex *Exchange) UpdateAgents(timeStep int) {
	ex.orderBook.askBook.SetBestData()
	ex.orderBook.bidBook.SetBestData()
	bestAsk := ex.orderBook.askBook.BestPrice
	bestBid := ex.orderBook.bidBook.BestPrice

	marketUpdate := common.MarketUpdate{
		TimeStep:  timeStep,
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

func (ex *Exchange) StartMarket(experimentID string) {
	log.WithFields(log.Fields{
		"Trading days":        ex.Info.TradingDays,
		"Training time steps": ex.Info.MarketEnd,
		"Num. Traders":        len(ex.agents),
		"ID":                  experimentID,
	}).Info("Market experiment started")

	for d := 0; d < ex.Info.TradingDays; d++ {
		// NOTE: clear orderbook at start of each day
		ex.orderBook.Reset()
		log.Info("Trading day:", d)
		for t := 0; t < ex.Info.MarketEnd; t++ {
			log.Info("Time-step:", t)
			// TODO: DISPATCH MARKET ORDER BASED ON SECHEDULE
			ok, order := ex.GetTraderOrder(t)
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
				}).Debug("No order was received this time step")
			}

			ex.MakeTrades(t)
			ex.UpdateAgents(t)
		}
		// TODO: store order books in database at the end of each day
		ex.orderBook.TradesToCSV(experimentID, d, ex.Info.MarketEnd)
	}

	// TODO: Persistent save of data

	log.WithFields(log.Fields{
		"Trades": len(ex.orderBook.tradeRecord),
		"Bids":   len(ex.orderBook.bidBook.Orders),
		"Asks":   len(ex.orderBook.askBook.Orders),
	}).Info("Experiment ended")
}
