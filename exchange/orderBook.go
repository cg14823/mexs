package exchange

import (
	"errors"
	"mexs/common"
	"sort"
)

//TODO: !!! IMPORTANT ORDERBOOK ASSUMES ONE ORDER PER TRADER AT ANY GIVEN TIME !!!

type OrderBookHalf struct {
	// bookType ASK or BID
	BookType string
	// Map of traderID to orders for now only one bid
	// per agent, this could be expanded using slices
	Orders map[int]*common.Order
	// Current best price negative values mean that they are uninitialized
	BestPrice int
	// traders offering best price uninitialized
	BestOrders []*common.Order
	// Max Depth ignored for now
	MaxDepth int
}

func (ob *OrderBookHalf) init(bookType string, maxDepth int) {
	ob.BookType = bookType
	ob.MaxDepth = maxDepth
	ob.Orders = make(map[int]*common.Order)
	ob.BestPrice = -1
	ob.BestOrders = make([]*common.Order, 0)
}

func (ob *OrderBookHalf) AddOrder(order *common.Order) error {
	if !order.IsValid() || order.OrderType == "NAN" {
		return errors.New("order could not be added")
	}

	if order.OrderType != ob.BookType {
		return errors.New("order and book type do not match")
	}

	ob.Orders[order.TraderID] = order
	return nil
}

func (ob *OrderBookHalf) RemoveOrder(order *common.Order) error {
	if !order.IsValid() || order.OrderType == "NAN" {
		return errors.New("NAN order can not be removed")
	}

	if order.OrderType != ob.BookType {
		return errors.New("order and book type do not match")
	}

	delete(ob.Orders, order.TraderID)
	return nil
}

func (ob *OrderBookHalf) GetBestOrder() ([]*common.Order, error) {
	if len(ob.Orders) == 0 {
		return make([]*common.Order, 0), nil
	}

	bestOrder := make([]*common.Order, 0)
	if ob.BookType == "ASK" {
		// int32 max value, exchange max ask should be smaller than this value
		currentBestPrice := 2147483647
		// There is a chance that more than one trader puts in same order if
		// there is no shout improvement or minimum increment

		for _, v := range ob.Orders {
			if v.Price < currentBestPrice {
				bestOrder = []*common.Order{v}
				currentBestPrice = v.Price
			} else if v.Price == currentBestPrice {
				bestOrder = append(bestOrder, v)
			}
		}
	} else if ob.BookType == "BID" {
		// all bids should be for a positive amount of money
		currentBestPrice := -1
		for _, v := range ob.Orders {
			if v.Price > currentBestPrice {
				bestOrder = []*common.Order{v}
				currentBestPrice = v.Price
			} else if v.Price == currentBestPrice {
				bestOrder = append(bestOrder, v)
			}
		}
	}

	if len(bestOrder) == 0 {
		return bestOrder, errors.New("no best order could be found for unknown reason")
	}

	return bestOrder, nil
}

func (ob *OrderBookHalf) SetBestData() error {
	orders, err := ob.GetBestOrder()

	if err != nil {
		return err
	}

	if len(orders) == 0 {
		return nil
	}

	ob.BestPrice = orders[0].Price
	sort.Sort(sort.Reverse(common.ByTimeStep(orders)))
	ob.BestOrders = orders
	return nil
}

type OrderBook struct {
	askBook     OrderBookHalf
	bidBook     OrderBookHalf
	tradeRecord []*common.Trade
}

func (ob *OrderBook) Init() {
	ob.askBook = OrderBookHalf{}
	ob.askBook.init("ASK", 100)
	ob.bidBook = OrderBookHalf{}
	ob.bidBook.init("BID", 100)
	ob.tradeRecord = make([]*common.Trade, 0)
}

func (ob *OrderBook) AddOrder(order *common.Order) error {

	if !order.IsValid() {
		return errors.New("order could not be added as it is not valid")
	}

	if order.OrderType == "NAN" {
		return errors.New("orders with NAN type can not be added")
	}

	if order.OrderType == "BID" {
		ob.bidBook.AddOrder(order)
		return nil
	}

	if order.OrderType == "ASK" {
		ob.askBook.AddOrder(order)
		return nil
	}

	return errors.New("unknown order type")
}

func (ob *OrderBook) RemoveOrder(order *common.Order) error {
	if !order.IsValid() {
		return errors.New("order could not be removed as it is not valid")
	}

	if order.OrderType == "NAN" {
		return errors.New("orders with NAN type can not be removed")
	}

	if order.OrderType == "BID" {
		ob.bidBook.RemoveOrder(order)
		return nil
	}

	if order.OrderType == "ASK" {
		ob.askBook.RemoveOrder(order)
		return nil
	}

	return errors.New("unknown order type cannot be removed")
}

func (ob *OrderBook) GetLastTrade() *common.Trade {
	if len(ob.tradeRecord) == 0 {
		return &common.Trade{TradeID: -1}
	}
	return ob.tradeRecord[len(ob.tradeRecord)-1]
}

func (ob *OrderBook) FindPossibleTrade() (trade bool, bid, ask *common.Order, err error) {
	askError := ob.askBook.SetBestData()
	if askError != nil {
		return false, &common.Order{}, &common.Order{}, askError
	}

	bidError := ob.bidBook.SetBestData()
	if bidError != nil {
		return false, &common.Order{}, &common.Order{}, bidError
	}

	if ob.bidBook.BestPrice < 0 && ob.askBook.BestPrice < 0 {
		return false, &common.Order{}, &common.Order{}, nil
	}

	if ob.bidBook.BestPrice >= ob.askBook.BestPrice {
		// Possible trade can be made
		return true, ob.bidBook.BestOrders[0], ob.askBook.BestOrders[0], nil
	}

	return false, &common.Order{}, &common.Order{}, nil
}
