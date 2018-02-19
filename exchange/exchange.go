package exchange

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"mexs/bots"
	"mexs/common"
	"time"
)

// AuctionParameters are the ones to be evolved by the GA
type AuctionParameters struct {
	// BidAskRatio is the proportion buyers to sellers
	BidAskRatio float32
	// k coefficient in k pricing rule pF = k *pB + (1-k)pA
	KPricing float32
	// The minimum increment in the next bid
	// If it is 0 it means there is no shout/spread improvement
	MinIncrement int
	// MaxShift is the maximum percentage a trader can move the current price
	MaxShift float32
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
	price := ex.GAVector.KPricing*bid.Price +
		(1-ex.GAVector.KPricing)*ask.Price

	return &common.Trade{
		TradeID: ex.orderBook.GetNextTradeID(),
		Price: price,
		BuyOrder: bid,
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
		"BuyerID": bid.TraderID,
		"SellerID": ask.TraderID,
		"Price": trade.Price,
	}).Info("Trade made!")
}

func (ex *Exchange) UpdateAgents(timeStep int) {
	ex.orderBook.askBook.SetBestData()
	ex.orderBook.bidBook.SetBestData()
	bestAsk := ex.orderBook.askBook.BestPrice
	bestBid := ex.orderBook.bidBook.BestPrice

	marketUpdate := common.MarketUpdate{
		TimeStep: timeStep,
		BestAsk: bestAsk,
		BestBid: bestBid,
		Bids: ex.orderBook.bidBook.OrdersToList(),
		Asks: ex.orderBook.askBook.OrdersToList(),
		Trades: ex.orderBook.tradeRecord,
	}

	for _, agent := range ex.agents {
		agent.MarketUpdate(marketUpdate)
	}
}

func (ex *Exchange) StartExperiment() {
	experimentID := uuid.New()

	log.WithFields(log.Fields{
		"Trading days":  ex.Info.TradingDays,
		"Training days": ex.Info.MarketEnd,
		"Num. Traders":  len(ex.agents),
		"ID":            experimentID,
	}).Info("Market experiment started")

	rand.Seed(time.Now().UTC().UnixNano())
	for d := 0; d < ex.Info.TradingDays; d++ {
		// NOTE: clear orderbook at start of each day
		ex.orderBook = OrderBook{}
		ex.orderBook.Init()
		log.Info("Trading day:", d)
		for t := 0; t < ex.Info.MarketEnd; t++ {
			log.Info("Time-step:", t)
			log.WithFields(log.Fields{
				"BestAsk": ex.orderBook.askBook.BestPrice,
				"BestBid": ex.orderBook.bidBook.BestPrice,
			}).Debug("Best at time-step:", t)
			traderIndex := rand.Intn(ex.AgentNum)
			var agent bots.RobotTrader = ex.agents[traderIndex]
			order := agent.GetOrder(t)
			// TODO: after getting order it is still needed to validated that it follow all rules
			log.WithFields(log.Fields{
				"TID":   order.TraderID,
				"Type":  order.OrderType,
				"Price": order.Price,
			}).Info("Order received")
			if order.OrderType == "NAN" {
				continue
			}

			err := ex.orderBook.AddOrder(order)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warn("Order Could not be added!")
			}
			ex.MakeTrades(t)
			ex.UpdateAgents(t)

		}
		// TODO: store order books in database at the end of each day
	}

	// TODO: Persistent save of data

	log.WithFields(log.Fields{
		"Trades": len(ex.orderBook.tradeRecord),
		"Bids": len(ex.orderBook.bidBook.Orders),
		"Asks": len(ex.orderBook.askBook.Orders),
	}).Info("Experiment ended")
}
