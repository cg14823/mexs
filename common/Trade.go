package common

import (
	"math"
	"time"
)

// This is other parameters from the market
// It should be change to use time.Time for async version
type MarketInfo struct {
	MaxPrice     float64 `json:"MaxPrice"`
	MinPrice     float64 `json:"MinPrice"`
	MinIncrement float64 `json:"MinIncrement"`
	// MarketEnd defines the time step at which the market ends
	MarketEnd int `json:"MarketEnd"`
	// Number of trading days
	TradingDays int `json:"TradingDays"`
}

//TODO: CHECK IF this is all that is needed
type MarketUpdate struct {
	TimeStep  int
	Day       int
	EID       string
	BestAsk   float64
	BestBid   float64
	Bids      []*Order
	Asks      []*Order
	Trades    []*Trade
	LastTrade *Trade
}

type Order struct {
	TraderID int
	// Order types: [Bid, ask, NAN, NA]  NA stands for non active it will
	OrderType string
	Price     float64
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
		o.Price >= ^ 0 && o.TimeStep >= 0 &&
		o.Quantity > 0 {
		return true
	} else if o.TraderID >= 0 && (o.OrderType == "NAN" || o.OrderType == "NA") {
		return true
	}

	return false
}

type Trade struct {
	TradeID   int
	BuyOrder  *Order
	SellOrder *Order
	BLimit float64
	SLimit float64
	Price     float64
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

// Round returns the nearest integer, rounding half away from zero.
//
// Special cases are:
//	Round(±0) = ±0
//	Round(±Inf) = ±Inf
//	Round(NaN) = NaN
func Round(x float64) float64 {
	t := math.Trunc(x)
	if math.Abs(x-t) >= 0.5 {
		return t + math.Copysign(1, x)
	}
	return t
}
