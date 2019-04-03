package p2pb2b

import (
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"testing"
)

const (
	apiKey0    = "a9c0c7780b9fab752bb88d40b542f0a1"
	apiSecret0 = "307be254ac040b0e7ccb045cb0fa2ccb"
	apiKey1    = "7e1669810e9050d27d3acf43ba9d1a17"
	apiSecret1 = "d092ec655aeb914fa48c65a2bc528feb"
	apiKey2    = "b3ac87f45c1e7818ae47aa247905f6dd"
	apiSecret2 = "61a9ea79d339c963af0dcdf35d1c2453"
	apiKey3    = "24d59fa9866e79262637c6115052b84f"
	apiSecret3 = "e8ea3a7120c11c2c3264c8069fcf5c88"
	apiKey4    = "2a1b15f7cc4639cc743d275c0490e272"
	apiSecret4 = "78ea30b852957e728f6e7a44555ef440"
)

func TestP2B(t *testing.T) {
	fmt.Println("tests run")
	p2b, err := MakeP2PB2BExchange(
		[]api.ExchangeAPIKey{
			{Key: apiKey0, Secret: apiSecret0},
			{Key: apiKey1, Secret: apiSecret1},
			{Key: apiKey2, Secret: apiSecret2},
			{Key: apiKey3, Secret: apiSecret3},
			{Key: apiKey4, Secret: apiSecret4},
		},
		false)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("getAccountBalances", func(t *testing.T) {
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

