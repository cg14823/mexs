package common

import "time"

// This is other parameters from the market
type MartketInfo struct {
	MaxPrice int
	MinPrice int
	TimeStep int
	BestBid  int
	BestAsk  int
}

type Order struct {
	TraderID int
	// Order types: [Bid, ask NAN]
	OrderType string
	Price     int
	Quantity  int
	TimeStep  int
	Time      time.Time
}

func (o *Order) isValid() bool {
	if o.TraderID > 0 &&
		(o.OrderType == "BID" || o.OrderType == "ASK") &&
		o.Price > 0 && o.TimeStep > 0 && o.Time.After(time.Now()) &&
		o.Quantity > 0 {
		return true
	} else if o.TraderID > 0 && o.OrderType == "NAN" {
		return true
	}

	return false
}

type Trade struct {
	TradeID   int
	BuyOrder  Order
	SellOrder Order
	Price     int
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
