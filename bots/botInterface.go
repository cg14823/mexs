package bots

import (
	"mexs/common"
)

type RobotCore struct {
	TraderID      int
	Type          string
	SellerOrBuyer string
	Balance       string
	Orders        []*TraderOrder
	TradeRecord   []*common.Trade
	MarketInfo    common.MartketInfo
	ActiveOrders  map[int]*common.Order
}

// TraderOrders encapsulate what the traders are supposed to do
type TraderOrder struct {
	LimitPrice int
	Quantity   int
	// Type should be BID or ASK
	Type string
}

func (to *TraderOrder) IsValid() bool {
	return to.LimitPrice > 0 && to.Quantity > 0 &&
		(to.Type == "BID" || to.Type == "ASK")
}

func (to *TraderOrder) IsBid() bool {
	return to.Type == "BID"
}

func (to *TraderOrder) IsAsk() bool {
	return to.Type == "ASK"
}

type RobotTrader interface {
	GetOrder(timeStep int) *common.Order
	MarketUpdate()
	TradeMade(trade common.Trade) bool
	AddOrder(order *TraderOrder)
	// Remove first Order
	RemoveOrder() error
}
