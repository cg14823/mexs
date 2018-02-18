package bots

import (
	"errors"
	"math/rand"
	"mexs/common"
	"time"
)

type ZICTrader struct {
	Info *RobotCore
}

func (t *ZICTrader) InitRobotCore(id int, algo string, sellerOrBuyer string, marketInfo common.MarketInfo) {
	t.Info = &RobotCore{
		TraderID:        id,
		Type:            algo,
		SellerOrBuyer:   sellerOrBuyer,
		ExecutionOrders: []*TraderOrder{},
		MarketInfo:      marketInfo,
		ActiveOrders:    map[int]*common.Order{},
		Balance:         0,
	}
	return
}

func (t *ZICTrader) GetOrder(timeStep int) *common.Order {
	if len(t.Info.ExecutionOrders) == 0 {
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	var order = t.Info.ExecutionOrders[0]
	if !order.IsValid() {
		err := t.RemoveOrder()
		if err != nil {
			//TODO: LOG ERROR HERE
		}

		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	if order.IsBid() {
		marketOrder := &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: order.Type,
			Price: rand.Intn(order.LimitPrice-t.Info.MarketInfo.MinPrice) +
				t.Info.MarketInfo.MinPrice,
			// For now just bid for all the quantity in order, no partitioning
			Quantity: order.Quantity,
			TimeStep: timeStep,
			Time:     time.Now(),
		}

		t.Info.ActiveOrders[timeStep] = marketOrder
		return marketOrder
	}

	marketOrder := &common.Order{
		TraderID:  t.Info.TraderID,
		OrderType: order.Type,
		Price: rand.Intn(t.Info.MarketInfo.MinPrice-order.LimitPrice) +
			order.LimitPrice,
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

func (t *ZICTrader) RemoveOrder() error {
	if len(t.Info.ExecutionOrders) == 0 {
		return errors.New("no order to be removed")
	}

	t.Info.ExecutionOrders = t.Info.ExecutionOrders[:len(t.Info.ExecutionOrders)-1]
	return nil
}

func (t *ZICTrader) TradeMade(trade *common.Trade) bool {
	// TODO: If the trader can have multiple Bids or offers
	// ... check that trade is still active using
	// ... the Active orders IGNORE FOR NOW

	t.Info.TradeRecord = append(t.Info.TradeRecord, trade)
	// TODO: For now assume that the is only one order thus trade ...
	//  ... must match first order in order array..
	// ... this could cause problems in async so should be changed in
	// ... the future

	// TODO: This code also assumes quantity matches, if it does not it
	// ... should update execute order for correct quantity and limit price

	t.Info.Balance += t.Info.ExecutionOrders[0].LimitPrice - trade.Price
	t.RemoveOrder()

	// In case trade is no longer possible this should return false
	// May be necessary for async markets
	return true
}

func (t *ZICTrader) MarketUpdate(info *common.MarketUpdate) {
	// ZIC trader does not care about market data
	return
}

// Check robot interface correctly implemented
var _ RobotTrader = (*ZICTrader)(nil)
