package main

import (
	"mexs/bots"
	"mexs/common"
	"fmt"
)

func main() {
	agents  :=  make(map[int]bots.RobotTrader)
	mInfo := common.MarketInfo{
		MaxPrice: float32(100),
		MinPrice: float32(1),
		MarketEnd: 30,
		TradingDays: 5,
	}
	z := bots.ZICTrader{}
	agents[1] = &z

	fmt.Println("HERE")
	z.InitRobotCore(1, "ZIC", "SELLER", mInfo)
	z.AddOrder(&bots.TraderOrder{
		LimitPrice: float32(10),
		Quantity: 1,
		Type: "BID",
	})

	var order *common.Order
	for _, v := range agents {
		order = v.GetOrder(1)
	}
	fmt.Println(order)
	fmt.Println(agents[1])
	fmt.Println(z)

}
