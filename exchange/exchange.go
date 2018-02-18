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
	ex.agents = make(map[int]bots.RobotTrader)
	ex.AgentNum = 0
}

func (ex *Exchange) SetTraders(traders map[int]bots.RobotTrader) {
	ex.agents = traders
	ex.AgentNum = len(traders)
}

func (ex *Exchange) PriceMatch (bid, ask *common.Order) *common.Trade{
	// TODO: Finish this function
	return &common.Trade{}
}

func (ex *Exchange) MakeTrades() {
	// TODO: FINISH this function
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
		log.Info("Trading day %d", d)
		for t := 0; t < ex.Info.MarketEnd; t++ {
			log.Info("Time-step: %d", t)

			traderIndex := rand.Intn(ex.AgentNum)
			var agent bots.RobotTrader = ex.agents[traderIndex]
			order := agent.GetOrder(t)

			log.WithFields(log.Fields{
				"TID": order.TraderID,
				"TYPE": order.OrderType,
				"PRICE": order.Price,
			}).Info("Order received")

			if order.OrderType != "NAN" {
				err := ex.orderBook.AddOrder(order)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Warn("Order Could not be added!")
				}
			}

		}
	}
}
