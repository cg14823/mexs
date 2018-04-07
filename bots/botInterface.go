package bots

import (
	"mexs/common"
)

type RobotCore struct {
	TraderID int
	// Trade algorithm used for now options are
	// ZIC
	Type string
	// This is used for the simple case where robot can
	// only be seller or a buyer but not change
	// to allow change we use TradeOrder.Type
	// for a seller all TradeOrder.Type = "ASK"
	// for buyer all TradeOrder.Type = "BID" in simple case
	SellerOrBuyer string
	// Orders to be executed by agent
	ExecutionOrders []*TraderOrder
	// Trades performed by that agent
	TradeRecord []*common.Trade
	// Stores market information
	MarketInfo common.MarketInfo
	// Orders currently in the market by the agents
	// Unused for know
	ActiveOrders map[int]*common.Order
	// Balance is money made by the agent
	// Balance = LimitPrice - transaction price
	Balance float64

	EID string
}

// TraderOrders encapsulate what the traders are supposed to do
type TraderOrder struct {
	LimitPrice float64
	// Quantity for now will be set to 1 it can be change
	// for more complicated simulations
	Quantity int
	// Type should be BID or ASK
	Type string
}

func (to *TraderOrder) IsValid() bool {
	return to.LimitPrice > 0.0 && to.Quantity > 0 &&
		(to.Type == "BID" || to.Type == "ASK")
}

func (to *TraderOrder) IsBid() bool {
	return to.Type == "BID"
}

func (to *TraderOrder) IsAsk() bool {
	return to.Type == "ASK"
}

type RobotTrader interface {
	InitRobotCore(id int, sellerOrBuyer string, marketInfo common.MarketInfo)
	SetOrders(orders []*TraderOrder)
	// Append execution order to array
	AddOrder(order *TraderOrder)
	// Remove first Order
	RemoveOrder() error
	// Should return their limit prices for logging reasons
	TradeMade(trade *common.Trade) (bool, float64)
	MarketUpdate(update common.MarketUpdate)
	GetOrder(timeStep int) *common.Order
	GetExecutionOrder() []*TraderOrder
	LogBalance(fileName string, day int, trade *common.Trade)
	LogOrder(fileName string, d, ts, tradeID int, tPrice float64)
}
