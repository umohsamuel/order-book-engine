package main

import (
	"fmt"
	"time"
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

func main() {
	fmt.Println("hello world")
}
