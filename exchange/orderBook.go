package exchange

import "mexs/common"

type OrderBookHalf struct {
	// bookType ask or bid
	bookType string
	// Map of tradeID to orders
	orders map[int]common.Order
	// Current best price negative values mean that they are uninitialized
	bestPrice int
	// trader offering best price uninitialized
	bestTID int
	// Max Depth
	maxDepth int
}

type OrderBook struct {
	askBook     OrderBookHalf
	bidBook     OrderBookHalf
	tradeRecord []*common.Trade
}

func (ob *OrderBook) AddOrder(order common.Order) {

}

func (ob *OrderBook) RemoveOrder(order common.Order) {

}

func (ob *OrderBook) GetLastTrade() common.Trade {
	return common.Trade{TradeID: -1}
}
