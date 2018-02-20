package exchange

import (
	"encoding/csv"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mexs/common"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

//NOTE: !!! IMPORTANT ORDERBOOK ASSUMES ONE ORDER PER TRADER AT ANY GIVEN TIME !!!

type OrderBookHalf struct {
	// bookType ASK or BID
	BookType string
	// Map of traderID to orders for now only one bid
	// per agent, this could be expanded using slices
	Orders map[int]*common.Order
	// Current best price negative values mean that they are uninitialized
	BestPrice float32
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

func (ob *OrderBookHalf) OrdersToList() []*common.Order {
	orders := []*common.Order{}
	for _, k := range ob.Orders {
		orders = append(orders, k)
	}
	return orders
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
	ob.BestPrice = -1
	ob.BestOrders = []*common.Order{}

	if len(ob.Orders) == 0 {
		return make([]*common.Order, 0), nil
	}

	bestOrder := make([]*common.Order, 0)
	if ob.BookType == "ASK" {
		// exchange max ask should be smaller than this value
		var currentBestPrice float32 = 1000000
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
		var currentBestPrice float32 = -1
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
		log.WithFields(log.Fields{
			"OrderBookType": ob.BookType,
		}).Error("Error ocurred:", err)
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
	lastTrade   *common.Trade
}

func (ob *OrderBook) Reset() {
	ob.askBook = OrderBookHalf{}
	ob.askBook.init("ASK", 100)
	ob.bidBook = OrderBookHalf{}
	ob.bidBook.init("BID", 100)
	ob.tradeRecord = make([]*common.Trade, 0)
}

func (ob *OrderBook) Init() {
	ob.askBook = OrderBookHalf{}
	ob.askBook.init("ASK", 100)
	ob.bidBook = OrderBookHalf{}
	ob.bidBook.init("BID", 100)
	ob.tradeRecord = make([]*common.Trade, 0)
	ob.lastTrade = &common.Trade{
		TradeID: -1,
	}
}

func (ob *OrderBook) AddOrder(order *common.Order) error {

	if !order.IsValid() {
		return errors.New(fmt.Sprintf("order could not be added as it is not valid: %#v", order))
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
		return ob.bidBook.RemoveOrder(order)
	}

	if order.OrderType == "ASK" {

		return ob.askBook.RemoveOrder(order)
	}

	return errors.New("unknown order type cannot be removed")
}

func (ob *OrderBook) GetLastTrade() *common.Trade {
	if len(ob.tradeRecord) == 0 {
		return &common.Trade{TradeID: -1}
	}
	return ob.tradeRecord[len(ob.tradeRecord)-1]
}

func (ob *OrderBook) GetNextTradeID() int {
	return len(ob.tradeRecord)
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

	if ob.bidBook.BestPrice < 0 || ob.askBook.BestPrice < 0 {
		return false, &common.Order{}, &common.Order{}, nil
	}

	if len(ob.bidBook.BestOrders) == 0 || len(ob.askBook.BestOrders) == 0 {
		return false, &common.Order{}, &common.Order{}, nil
	}

	if ob.bidBook.BestPrice >= ob.askBook.BestPrice {
		// Possible trade can be made and is in a FIFO basis
		return true, ob.bidBook.BestOrders[0], ob.askBook.BestOrders[0], nil
	}

	return false, &common.Order{}, &common.Order{}, nil
}

func (ob *OrderBook) RecordTrade(trade *common.Trade) error {
	err := ob.RemoveOrder(trade.BuyOrder)
	if err != nil {
		log.WithFields(log.Fields{
			"order": trade.BuyOrder,
		}).Error("can not remove order so trade not made")
		return err
	}

	err = ob.RemoveOrder(trade.SellOrder)
	if err != nil {
		log.WithFields(log.Fields{
			"order": trade.SellOrder,
		}).Error("can not remove order so trade not made")
		return err
	}

	ob.tradeRecord = append(ob.tradeRecord, trade)
	ob.lastTrade = trade
	return nil
}

func (ob *OrderBook) TradesToCSV(experimentID string, tradingDay, maxTD int) {
	// CSV file format is as follows TradeID,TradingDay,TimeStep,Price,SellerID,BuyerID,SellerPrice,BuyerPrice
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/TRADES_ID-%s_%d-%d.csv", experimentID, tradingDay, maxTD))
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  tradingDay,
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  tradingDay,
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("Trade CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"ID", "TradingDay", "TimeStep", "Price", "SellerID", "BuyerID", "AskPrice", "BidPrice"})
	for _, trade := range ob.tradeRecord {
		row := []string{
			strconv.Itoa(trade.TradeID),
			strconv.Itoa(tradingDay),
			strconv.Itoa(trade.TimeStep),
			fmt.Sprintf("%.3f", trade.Price),
			strconv.Itoa(trade.SellOrder.TraderID),
			strconv.Itoa(trade.BuyOrder.TraderID),
			fmt.Sprintf("%.3f", trade.SellOrder.Price),
			fmt.Sprintf("%.3f", trade.BuyOrder.Price),
		}
		writer.Write(row)
	}

	log.Debug("Trades saved to file:", fileName)
}

// TODO: IMPLEMENT BID TO CSV and ASK to CSV
