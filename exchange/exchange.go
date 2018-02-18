package exchange

import "mexs/bots"

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
type Exchange interface {
	StartUp(id int, auctionParameters AuctionParameters)
	StartOrderBook()
	UpdateTraders([]*bots.RobotTrader)
}
