package p2pb2b

import (
	"fmt"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"testing"
)

const (
	apiKey0    = "b97cd1a2d30a83ba8417994117c5c78d"
	apiSecret0 = "f8b1e34e77c3d048efe6b2a1117f1646"
	apiKey1    = "c339c96cd4bdb3596d404cadde8746ba"
	apiSecret1 = "988191e11242cafd7514fb2d84d01ceb"
	apiKey2    = "95733fc9a4971707fba0a8c215f57740"
	apiSecret2 = "6f5cb056eb57f269dc6e15caf1dc6c08"
)

func TestP2B(t *testing.T) {
	p2b, err := MakeP2PB2BExchange(
		[]api.ExchangeAPIKey{
			{Key: apiKey0, Secret: apiSecret0},
			//{Key: apiKey1, Secret: apiSecret1},
			//{Key: apiKey2, Secret: apiSecret2},
		},
		false)
	if err != nil {
		fmt.Println(err)
		t.Fatal(err)
	}

	t.Run("getAccountBalancaes", func(t *testing.T) {
		getAccountBalances(t, p2b.(*pbExchange))
	})

	t.Run("getTickerPrice", func(t *testing.T) {
		getTickerPrice(t, p2b.(*pbExchange))
	})

	t.Run("getOrderBook", func(t *testing.T) {
		getOrderBook(t, p2b.(*pbExchange))
	})

	t.Run("addOrder", func(t *testing.T) {
		addOrder(t, p2b.(*pbExchange))
	})

	t.Run("cancelOpenOrder", func(t *testing.T) {
		cancelOpenOrder(t, p2b.(*pbExchange))
	})

}

func getAccountBalances(t *testing.T, p2b *pbExchange) {
	balances, err := p2b.GetAccountBalances([]interface{}{model.Asset("USD"), model.Asset("XLM"), model.Asset("BTC")})
	ok := err == nil && len(balances) == 3
	if !ok {
		t.Error(balances, err)
	}
}

func getTickerPrice(t *testing.T, p2b *pbExchange) {
	prices, err := p2b.GetTickerPrice([]model.TradingPair{
		{Base: "ETH", Quote: "BTC"},
		{Base: "XLM", Quote: "BTC"},
	})
	ok := err == nil && len(prices) == 2
	if !ok {
		t.Error(prices, err)
	}
}

func getOrderBook(t *testing.T, p2b *pbExchange) {
	book, err := p2b.GetOrderBook(&model.TradingPair{Base: "ETH", Quote: "BTC"}, 20)
	ok := err == nil && len(book.Asks()) == 20 && len(book.Bids()) == 20
	if !ok {
		t.Error(book, err)
	}
}

func addOrder(t *testing.T, p2b *pbExchange) {
	volume := model.NumberFromFloat(10, 8)
	price := model.NumberFromFloat(0.00003, 8)
	order := &model.Order{
		Pair:        &model.TradingPair{Base: "XLM", Quote: "BTC"},
		OrderAction: true, // sell
		OrderType:   1,
		Price:       price,
		Volume:      volume,
	}

	_, err := p2b.AddOrder(order)
	if err != nil {
		t.Error(err)
	}

}

func cancelOpenOrder(t *testing.T, p2b *pbExchange) {
	pair := &model.TradingPair{Base: "XLM", Quote: "BTC"}
	orders, err := p2b.GetOpenOrders([]*model.TradingPair{pair})
	ok := err == nil && len(orders[*pair]) > 0
	if !ok {
		t.Error(orders, err)
		return
	}
	id := model.TransactionID(orders[*pair][0].ID)
	result, err := p2b.CancelOrder(&id, *pair)
	ok = err == nil && result == model.CancelResultCancelSuccessful
	if !ok {
		t.Error(result, err)
	}
}
