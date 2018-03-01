package bots

import (
	"crypto/rand"
	"errors"
	log "github.com/sirupsen/logrus"
	"math/big"
	"mexs/common"
	"time"
	"os"
	"encoding/csv"
	"strconv"
	"fmt"
	"path/filepath"
)

type ZICTrader struct {
	Info RobotCore
}

func (t *ZICTrader) InitRobotCore(id int, sellerOrBuyer string, marketInfo common.MarketInfo) {
	t.Info = RobotCore{
		TraderID:        id,
		Type:            "ZIC",
		SellerOrBuyer:   sellerOrBuyer,
		ExecutionOrders: []*TraderOrder{},
		MarketInfo:      marketInfo,
		ActiveOrders:    map[int]*common.Order{},
		Balance:         0,
	}
}

func (t *ZICTrader) GetOrder(timeStep int) *common.Order {

	if len(t.Info.ExecutionOrders) == 0 {
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NA",
		}
	}

	var order = t.Info.ExecutionOrders[0]
	if !order.IsValid() {
		err := t.RemoveOrder()
		if err != nil {
			log.WithFields(log.Fields{
				"ExecOrder": order,
				"Place":     "ZIC Trader GetOrder",
			}).Error("Error:", err)
		}

		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	if order.IsBid() {
		bidPrice := order.LimitPrice - t.Info.MarketInfo.MinPrice
		if bidPrice > 0 {
			randPrice, _ := rand.Int(rand.Reader, big.NewInt(int64(bidPrice)))
			bidPrice = float64(randPrice.Int64()) + t.Info.MarketInfo.MinPrice
		}

		marketOrder := &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: order.Type,
			Price:     bidPrice,
			// For now just bid for all the quantity in order, no partitioning
			Quantity: order.Quantity,
			TimeStep: timeStep,
			Time:     time.Now(),
		}

		t.Info.ActiveOrders[timeStep] = marketOrder
		return marketOrder
	}

	bidPrice := t.Info.MarketInfo.MaxPrice - order.LimitPrice
	if bidPrice > 0 {
		randPrice, _ := rand.Int(rand.Reader, big.NewInt(int64(bidPrice)))
		bidPrice = float64(randPrice.Int64()) + order.LimitPrice
	}

	marketOrder := &common.Order{
		TraderID:  t.Info.TraderID,
		OrderType: order.Type,
		Price:     bidPrice,
		// For now just bid for all the quantity in order, no partitioning
		Quantity: order.Quantity,
		TimeStep: timeStep,
		Time:     time.Now(),
	}

	t.Info.ActiveOrders[timeStep] = marketOrder
	return marketOrder

}

func (t *ZICTrader) AddOrder(order *TraderOrder) {
	t.Info.ExecutionOrders = append(t.Info.ExecutionOrders, order)
	return
}

func (t *ZICTrader) SetOrders(orders []*TraderOrder) {
	t.Info.ExecutionOrders = orders
}

func (t *ZICTrader) RemoveOrder() error {
	if len(t.Info.ExecutionOrders) == 0 {
		return errors.New("no order to be removed")
	}

	t.Info.ExecutionOrders = t.Info.ExecutionOrders[:len(t.Info.ExecutionOrders)-1]
	return nil
}

func (t *ZICTrader) TradeMade(trade *common.Trade) bool {
	// If the trader can have multiple Bids or offers
	// ... check that trade is still active using
	// ... the Active orders IGNORE FOR NOW

	// For system with multiple order queueing and that allows
	// canceling this should be part of a second function such as confirm trade
	// ignore for now as it is async and without order queueing for the time being
	t.Info.TradeRecord = append(t.Info.TradeRecord, trade)
	// For now assume that the is only one order thus trade ...
	//  ... must match first order in order array..
	// ... this could cause problems in async so should be changed in
	// ... the future

	// This code also assumes quantity matches, if it does not it
	// ... should update execute order for correct quantity and limit price
	if trade.SellOrder.TraderID == t.Info.TraderID {
		t.Info.Balance += trade.Price - t.Info.ExecutionOrders[0].LimitPrice
	} else {
		t.Info.Balance += t.Info.ExecutionOrders[0].LimitPrice - trade.Price
	}

	t.RemoveOrder()

	// In case trade is no longer possible this should return false
	// May be necessary for async markets
	return true
}

func (t *ZICTrader) MarketUpdate(info common.MarketUpdate) {
	// ZIC trader does not care about market data
	return
}

func (t *ZICTrader) GetExecutionOrder() []*TraderOrder {
	return t.Info.ExecutionOrders
}

func (t *ZICTrader) LogBalance(fileName string, day int, trade *common.Trade) {
	fileName, err := filepath.Abs(fileName+"/ZICTradersLog.csv")
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  day,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}

	addHeader := true
	if _, err := os.Stat(fileName); err == nil {
		addHeader = false
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day":  day,
			"error":        err.Error(),
		}).Error("ZIC trader CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "TID", "TradeID", "Profit", "TPrice",})
	}

	writer.Write([]string{
		strconv.Itoa(day),
		strconv.Itoa(trade.TimeStep),
		strconv.Itoa(t.Info.TraderID),
		strconv.Itoa(trade.TradeID),
		fmt.Sprintf("%.5f",t.Info.Balance),
		fmt.Sprintf("%.5f",trade.Price),
	})
}

// Check robot interface correctly implemented
var _ RobotTrader = (*ZICTrader)(nil)
