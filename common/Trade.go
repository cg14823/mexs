package common

import (
	"time"
)

// This is other parameters from the market
// It should be change to use time.Time for async version
type MarketInfo struct {
	MaxPrice float32
	MinPrice float32
	// MarketEnd defines the time step at which the market ends
	MarketEnd int
	// Number of trading days
	TradingDays int
}

//TODO: CHECK IF this is all that is needed
type MarketUpdate struct {
	TimeStep int
	BestAsk  float32
	BestBid  float32
	Bids     []*Order
	Asks     []*Order
	Trades   []*Trade
}

type Order struct {
	TraderID int
	// Order types: [Bid, ask, NAN]
	OrderType string
	Price     float32
	Quantity  int
	TimeStep  int
	Time      time.Time
}

// For sorting lists of orders
type ByPrice []*Order
type ByTimeStep []*Order

func (o ByPrice) Len() int {
	return len(o)
}

func (o ByPrice) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o ByPrice) Less(i, j int) bool {
	return o[i].Price < o[j].Price
}

func (o ByTimeStep) Len() int {
	return len(o)
}

func (o ByTimeStep) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o ByTimeStep) Less(i, j int) bool {
	return o[i].TimeStep < o[j].TimeStep
}

func (o *Order) IsValid() bool {
	if o.TraderID >= 0 &&
		(o.OrderType == "BID" || o.OrderType == "ASK") &&
		o.Price > 0 && o.TimeStep >= 0 &&
		o.Quantity > 0 {
		return true
	} else if o.TraderID >= 0 && o.OrderType == "NAN" {
		return true
	}

	return false
}

type Trade struct {
	TradeID   int
	BuyOrder  *Order
	SellOrder *Order
	Price     float32
	Quantity  int
	TimeStep  int
	Time      time.Time
}

func (t *Trade) GetBuyer() int {
	return t.BuyOrder.TraderID
}
func (t *Trade) GetSeller() int {
	return t.BuyOrder.TraderID
}
