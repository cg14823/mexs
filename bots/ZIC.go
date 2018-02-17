package bots

import (
	"errors"
	"math/rand"
	"mexs/common"
	"time"
)

type ZICTrader struct {
	Info RobotCore
}

func (t *ZICTrader) GetOrder(timeStep int) *common.Order {
	if len(t.Info.Orders) == 0 {
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	var order = t.Info.Orders[0]
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

		t.Info.ActiveOrders[timeStep]= marketOrder
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

	t.Info.ActiveOrders[timeStep]= marketOrder
	return marketOrder

}

func (t *ZICTrader) RemoveOrder() error {
	if len(t.Info.Orders) == 0 {
		return errors.New("no order to be removed")
	}
	t.Info.Orders = t.Info.Orders[:len(t.Info.Orders)-1]
	return nil
}

func (t *ZICTrader) AddOrder(order *TraderOrder) {
	t.Info.Orders = append(t.Info.Orders, order)
}

func (t *ZICTrader) TradeMade(trade *common.Trade) {

	// TODO: If the trader can have multiple Bids or offers
	// TODO(Continued): check that trade is still active using
	// TODO(CONTINUED): the Active orders IGNORE FOR NOW

}
