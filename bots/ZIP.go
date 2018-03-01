package bots

// This trader is based on Dave Cliff's  1997 work, most of the logic has been ported
// from Dave Cliff's open source project 'BristolStockExchange' available at
// https://github.com/davecliff/BristolStockExchange, parts of it where modified to
// better fit this simulator, but it still makes the same decisions overall.

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"mexs/common"
	"sort"
	"time"
	"os"
	"encoding/csv"
	"strconv"
	"fmt"
	"path/filepath"
)

type ZIPTrader struct {
	Info             RobotCore
	job              *TraderOrder
	active           bool
	lastDelta        float64
	beta             float64
	momentum         float64
	margin           float64
	marginBuy        float64
	marginSell       float64
	ca               float64
	cr               float64
	price            float64
	prevBestBidPrice float64
	prevBestBidQty   int
	prevBestAskPrice float64
	prevBestAskQty   int
}

func (t *ZIPTrader) InitRobotCore(id int, sellerOrBuyer string, marketInfo common.MarketInfo) {
	t.Info = RobotCore{
		TraderID:        id,
		Type:            "ZIP",
		SellerOrBuyer:   sellerOrBuyer,
		ExecutionOrders: []*TraderOrder{},
		MarketInfo:      marketInfo,
		ActiveOrders:    map[int]*common.Order{},
		Balance:         0,
	}

	// Initialize ZIP parameters following Dave cliff 1997 paper procedure
	t.active = false
	t.lastDelta = 0.0
	t.beta = 0.1 + 0.4*rand.Float64()
	t.momentum = 0.1 * rand.Float64()
	t.ca = 0.05 // t.ca & .cr were hard-coded in '97 but parameterised later
	t.cr = 0.05
	t.marginBuy = -1.0 * (0.05 + 0.3*rand.Float64())
	t.marginSell = 0.05 + 0.3*rand.Float64()
	t.prevBestBidPrice = -1
	t.prevBestAskPrice = -1
	t.job = &TraderOrder{Type: "NA"}
}

func (t *ZIPTrader) SetOrders(orders []*TraderOrder) {
	t.Info.ExecutionOrders = orders
}

func (t *ZIPTrader) AddOrder(order *TraderOrder) {
	t.Info.ExecutionOrders = append(t.Info.ExecutionOrders, order)
}

func (t *ZIPTrader) RemoveOrder() error {
	if len(t.Info.ExecutionOrders) == 0 {
		return errors.New("no order to be removed")
	}

	t.Info.ExecutionOrders = t.Info.ExecutionOrders[:len(t.Info.ExecutionOrders)-1]
	return nil
}

func (t *ZIPTrader) GetOrder(timeStep int) *common.Order {
	if len(t.Info.ExecutionOrders) == 0 {
		t.active = false
		t.job = &TraderOrder{Type: "NA"}
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NA",
		}
	}

	order := t.Info.ExecutionOrders[0]
	if !order.IsValid() {
		err := t.RemoveOrder()
		if err != nil {
			log.WithFields(log.Fields{
				"ExecOrder": order,
				"Place":     "ZIP Trader GetOrder",
			}).Error("Error:", err)
		}

		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	t.active = true
	t.job = order
	if order.IsBid() {
		t.margin = t.marginBuy
	} else {
		t.margin = t.marginSell
	}

	quotePrice := common.Round(order.LimitPrice*1.0 + t.margin)
	marketOrder := &common.Order{
		TraderID:  t.Info.TraderID,
		OrderType: order.Type,
		Price:     quotePrice,
		Quantity:  order.Quantity,
		TimeStep:  timeStep,
		Time:      time.Now(),
	}
	t.Info.ActiveOrders[timeStep] = marketOrder
	return marketOrder
}

func (t *ZIPTrader) MarketUpdate(update common.MarketUpdate) {
	bidImproved := false
	bidHit := false
	if update.BestBid != -1 {
		// Non empty bid lob
		if t.prevBestBidPrice < update.BestBid {
			// best bid has improved
			// NOTE: NB doesn't check if the improvement was by self SHOULD IT?
			bidImproved = true
		} else if update.LastTrade.TimeStep == update.TimeStep &&
			t.prevBestBidPrice >= update.BestBid {
			bidHit = true
		}
	} else if t.prevBestBidPrice != -1 {
		bidHit = true
	}

	askImporved := false
	askLifted := false
	if update.BestAsk != -1 {
		// Non empty ask lob
		if t.prevBestAskPrice > update.BestAsk {
			askImporved = true
		} else if update.LastTrade.TimeStep == update.TimeStep &&
			t.prevBestAskPrice <= update.BestAsk {
			askLifted = true
		}
	} else if t.prevBestAskPrice != -1 {
		askLifted = true
	}

	//log.WithFields(log.Fields{
	//	"TraderID":     t.Info.TraderID,
	//	"Bid Improved": bidImproved,
	//	"Bid Hit":      bidHit,
	//	"Ask Improved": askImporved,
	//	"Ask Lifted":   askLifted,
	//}).Debug("ZIP trader update")

	deal := bidHit || askLifted

	if t.job.IsAsk() {
		if deal {
			if t.price <= update.LastTrade.Price {
				targetPrice := t.targetUp(update.LastTrade.Price)
				t.profitAlter(targetPrice)
			} else if askLifted && t.active && !t.willingToTrade(update.LastTrade.Price) {
				targetPrice := t.targetDown(update.LastTrade.Price)
				t.profitAlter(targetPrice)
			}
		} else {
			// no deal: aim for a target price higher than bid best bid
			if askImporved && t.price > update.BestAsk {
				if update.BestBid != -1 {
					targetPrice := t.targetUp(update.BestBid)
					t.profitAlter(targetPrice)
				} else {
					sort.Sort(common.ByPrice(update.Asks))
					t.profitAlter(update.Asks[0].Price)
				}
			}
		}
	} else if t.job.IsBid() {
		if deal {
			if t.price >= update.LastTrade.Price {
				targetPrice := t.targetDown(update.LastTrade.Price)
				t.profitAlter(targetPrice)
			} else if bidHit && t.active && !t.willingToTrade(update.LastTrade.Price) {
				targetPrice := t.targetUp(update.LastTrade.Price)
				t.profitAlter(targetPrice)
			}
		} else {
			if bidImproved && t.price < update.BestBid {
				if update.BestAsk != -1 {
					targetPrice := t.targetDown(update.BestAsk)
					t.profitAlter(targetPrice)
				} else {
					sort.Sort(common.ByPrice(update.Bids))
					t.profitAlter(update.Bids[0].Price)
				}
			}
		}
	}

	t.prevBestBidPrice = update.BestBid
	t.prevBestAskPrice = update.BestAsk

}

func (t *ZIPTrader) willingToTrade(price float64) bool {
	if t.job.IsBid() && t.active && t.price >= price {
		return true
	}

	if t.job.IsBid() && t.active && t.price <= price {
		return true
	}

	return false
}

func (t *ZIPTrader) targetUp(price float64) float64 {
	//  Generate a higher target price by randomly perturbing given price
	absolutePerturbation := t.ca * rand.Float64()
	relativePerturbation := t.price * (1.0 + (t.cr * rand.Float64()))
	target := absolutePerturbation + relativePerturbation
	return common.Round(target)
}

func (t *ZIPTrader) targetDown(price float64) float64 {
	//  Generate a lower target price by randomly perturbing given price
	absolutePerturbation := t.ca * rand.Float64()
	relativePerturbation := t.price * (1.0 + (t.cr * rand.Float64()))
	target := absolutePerturbation - relativePerturbation
	return common.Round(target)
}

func (t *ZIPTrader) profitAlter(price float64) {
	oldPrice := t.price
	diff := price - oldPrice
	change := ((1.0 - t.momentum) * (t.beta * diff)) + t.momentum*t.lastDelta
	t.lastDelta = change
	newMargin := ((t.price + change) / t.job.LimitPrice) - 1.0

	if t.job.IsBid() {
		if newMargin < 0.0 {
			t.marginBuy = newMargin
			t.margin = newMargin
		}
	} else {
		if newMargin > 0.0 {
			t.marginSell = newMargin
			t.margin = newMargin
		}
	}

	t.price = common.Round(t.job.LimitPrice * (1.0 + t.margin))
}

func (t *ZIPTrader) TradeMade(trade *common.Trade) bool {
	// Got to ZIC.go to read about this function weaknesses and reasoning
	t.Info.TradeRecord = append(t.Info.TradeRecord, trade)
	if trade.SellOrder.TraderID == t.Info.TraderID {
		t.Info.Balance += trade.Price - t.Info.ExecutionOrders[0].LimitPrice
	} else {
		t.Info.Balance += t.Info.ExecutionOrders[0].LimitPrice - trade.Price
	}
	t.RemoveOrder()
	return true
}
func (t *ZIPTrader) GetExecutionOrder() []*TraderOrder {
	return t.Info.ExecutionOrders
}

func (t *ZIPTrader) LogBalance(fileName string, day int, trade *common.Trade) {
	fileName, err := filepath.Abs(fileName+"/ZIPTradersLog.csv")
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
		}).Error("ZIP trader CSV file could not be made")
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

var _ RobotTrader = (*ZIPTrader)(nil)
