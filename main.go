package main

import (
	"fmt"
	"time"

	"github.com/google/btree"
)

type Order struct {
	OrderID   string
	Price     float64
	Quantity  float64
	Timestamp time.Time
	Side      string
}

type PriceLevel struct {
	Price  float64
	Orders []*Order
	Side   string
}

func (p PriceLevel) Less(than btree.Item) bool {
	o := than.(PriceLevel)
	if p.Side == "buy" {
		return p.Price > o.Price
	} else {
		return p.Price < o.Price
	}
}

type OrderBook struct {
	BuyTree         *btree.BTree
	SellTree        *btree.BTree
	Orders          map[string]*Order
	LastTradedPrice float64
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		BuyTree:  btree.New(3),
		SellTree: btree.New(3),
		Orders:   make(map[string]*Order),
	}
}

func (ob *OrderBook) AddOrder(order *Order) {
	ob.Orders[order.OrderID] = order

	priceLevel := PriceLevel{
		Price: order.Price,
		Side:  order.Side,
	}

	var tree *btree.BTree

	if order.Side == "buy" {
		tree = ob.BuyTree
	} else {
		tree = ob.SellTree
	}

	item := tree.Get(priceLevel)
	if item != nil {
		existingLevel := item.(PriceLevel)
		existingLevel.Orders = append(existingLevel.Orders, order)
		tree.ReplaceOrInsert(existingLevel)
	} else {
		priceLevel.Orders = []*Order{order}
		tree.ReplaceOrInsert(priceLevel)
	}
}

func (ob *OrderBook) RemoveOrder(orderID string) error {
	order, ok := ob.Orders[orderID]
	if !ok {
		return fmt.Errorf("order not found")
	}

	ob.removeOrder(order)
	delete(ob.Orders, orderID)
	return nil
}

func (ob *OrderBook) removeOrder(order *Order) {
	priceLevel := PriceLevel{
		Price: order.Price,
		Side:  order.Side,
	}

	var tree *btree.BTree
	if order.Side == "buy" {
		tree = ob.BuyTree
	} else {
		tree = ob.SellTree
	}

	item := tree.Get(priceLevel)
	if item != nil {
		existingLevel := item.(PriceLevel)
		for i, ord := range existingLevel.Orders {
			if ord.OrderID == order.OrderID {
				existingLevel.Orders = append(existingLevel.Orders[:i], existingLevel.Orders[i+1:]...)

				if len(existingLevel.Orders) == 0 {
					tree.Delete(existingLevel)
				} else {
					tree.ReplaceOrInsert(existingLevel)
				}
				break
			}
		}
	}
}

func (ob *OrderBook) ModifyOrderSide(orderID string, newSide string) error {
	if newSide != "buy" && newSide != "sell" {
		return fmt.Errorf("invalid side")
	}

	order, exists := ob.Orders[orderID]
	if !exists {
		return fmt.Errorf("order not found")
	}

	ob.removeOrder(order)

	order.Side = newSide

	ob.AddOrder(order)
	return nil
}

func (ob *OrderBook) ModifyOrderQuantity(orderID string, newQuantity float64) error {
	if newQuantity < 0 {
		return fmt.Errorf("invalid quantity")
	}

	order, exists := ob.Orders[orderID]
	if !exists {
		return fmt.Errorf("order not found")
	}

	if newQuantity == 0 {
		ob.removeOrder(order)
		delete(ob.Orders, orderID)
	} else {
		order.Quantity = newQuantity
	}
	return nil
}

func (ob *OrderBook) ModifyOrderPrice(orderID string, newPrice float64) error {
	order, exists := ob.Orders[orderID]
	if !exists {
		return fmt.Errorf("order not found")
	}

	ob.removeOrder(order)

	order.Price = newPrice

	ob.AddOrder(order)
	return nil
}

func (ob *OrderBook) MatchOrders() {
	for {
		buyItem := ob.BuyTree.Min()
		sellItem := ob.SellTree.Min()
		if buyItem == nil || sellItem == nil {
			break
		}

		highestBuy := buyItem.(PriceLevel)
		lowestSell := sellItem.(PriceLevel)

		if highestBuy.Price >= lowestSell.Price {
			buyOrder := highestBuy.Orders[0]
			sellOrder := lowestSell.Orders[0]

			tradePrice := sellOrder.Price

			ob.LastTradedPrice = tradePrice

			tradeQuantity := min(buyOrder.Quantity, sellOrder.Quantity)

			fmt.Printf("Trade executed: Buy Order %s and Sell Order %s at Price %.2f for Quantity %.2f\n",
				buyOrder.OrderID, sellOrder.OrderID, tradePrice, tradeQuantity)

			buyOrder.Quantity -= tradeQuantity
			sellOrder.Quantity -= tradeQuantity

			if buyOrder.Quantity == 0 {
				ob.removeOrder(buyOrder)
				delete(ob.Orders, buyOrder.OrderID)
			}

			if sellOrder.Quantity == 0 {
				ob.removeOrder(sellOrder)
				delete(ob.Orders, sellOrder.OrderID)
			}
		} else {
			break
		}
	}
}

func (ob *OrderBook) GetBestBid() (float64, bool) {
	if ob.BuyTree.Len() == 0 {
		return 0, false
	}
	item := ob.BuyTree.Min()
	bestBidItem := item.(PriceLevel)

	return bestBidItem.Price, true
}

func (ob *OrderBook) GetBestAsk() (float64, bool) {
	if ob.SellTree.Len() == 0 {
		return 0, false
	}
	item := ob.SellTree.Min()
	bestAskItem := item.(PriceLevel)

	return bestAskItem.Price, true
}

func (ob *OrderBook) GetMidPrice() (float64, bool) {
	bestBid, hasBid := ob.GetBestBid()
	bestAsk, hasAsk := ob.GetBestAsk()
	if hasBid && hasAsk {
		return (bestBid + bestAsk) / 2, true
	}

	return 0, false
}

func (ob *OrderBook) GetCurrentMarketPrice() (float64, bool) {
	if ob.LastTradedPrice != 0 {
		return ob.LastTradedPrice, true
	}
	midPrice, ok := ob.GetMidPrice()
	if ok {
		return midPrice, true
	}

	return 0, false
}

func (ob *OrderBook) DisplayOrderBook() {
	fmt.Println("Order Book:")
	fmt.Println("Buy Orders:")
	ob.BuyTree.Ascend(func(i btree.Item) bool {

		item := i.(PriceLevel)
		for _, ord := range item.Orders {

			fmt.Printf("OrderID: %s, Price: %.2f, Quantity: %.2f\n", ord.OrderID, ord.Price, ord.Quantity)
		}

		return true
	})

	fmt.Println("Sell Orders:")
	ob.SellTree.Ascend(func(i btree.Item) bool {
		item := i.(PriceLevel)
		for _, ord := range item.Orders {
			fmt.Printf("OrderID: %s, Price: %.2f, Quantity: %.2f\n", ord.OrderID, ord.Price, ord.Quantity)
		}

		return true
	})

	fmt.Println("------------------------------")
}

func main() {
	orderBook := NewOrderBook()

	order1 := &Order{
		OrderID:   "B1",
		Price:     100.0,
		Quantity:  10,
		Timestamp: time.Now(),
		Side:      "buy",
	}
	orderBook.AddOrder(order1)

	order2 := &Order{
		OrderID:   "S1",
		Price:     105.0,
		Quantity:  5,
		Timestamp: time.Now(),
		Side:      "sell",
	}
	orderBook.AddOrder(order2)

	order3 := &Order{
		OrderID:   "B2",
		Price:     102.0,
		Quantity:  7,
		Timestamp: time.Now(),
		Side:      "buy",
	}
	orderBook.AddOrder(order3)

	order4 := &Order{
		OrderID:   "S2",
		Price:     99.0,
		Quantity:  8,
		Timestamp: time.Now(),
		Side:      "sell",
	}
	orderBook.AddOrder(order4)

	orderBook.DisplayOrderBook()

	orderBook.MatchOrders()

	marketPrice, ok := orderBook.GetCurrentMarketPrice()
	if ok {
		fmt.Printf("Current Market Price: %.2f\n", marketPrice)
	} else {
		fmt.Println("Market price is not available.")
	}

	orderBook.DisplayOrderBook()

	err := orderBook.ModifyOrderPrice("B2", 98.0)
	if err != nil {
		fmt.Println(err)
	}

	err = orderBook.ModifyOrderSide("S1", "buy")
	if err != nil {
		fmt.Println(err)
	}

	err = orderBook.ModifyOrderQuantity("B1", 0)
	if err != nil {
		fmt.Println(err)
	}

	orderBook.DisplayOrderBook()

	orderBook.MatchOrders()

	marketPrice, ok = orderBook.GetCurrentMarketPrice()
	if ok {
		fmt.Printf("Current Market Price: %.2f\n", marketPrice)
	} else {
		fmt.Println("Market price is not available.")
	}

	orderBook.DisplayOrderBook()
}
