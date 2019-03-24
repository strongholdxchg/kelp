package p2pb2b

import (
	"errors"
	"fmt"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ensure that pbExchange conforms to the Exchange interface
var _ api.Exchange = &pbExchange{}
var ErrorNotSupported = errors.New("FUNCTION_NOT_SUPPORTED")

const precisionBalances = 8

// pbExchange is the implementation for the p2pb2b Exchange
type pbExchange struct {
	assetConverter *model.AssetConverter
	apis           []*P2BApi
	apiNextIndex   uint8
	delimiter      string
	isSimulated    bool // will simulate add and cancel orders if this is true
}

// makepbExchange is a factory method to make the pb exchange
func MakeP2PB2BExchange(apiKeys []api.ExchangeAPIKey, isSimulated bool) (api.Exchange, error) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)

	if len(apiKeys) == 0 || len(apiKeys) > math.MaxUint8 {
		return nil, fmt.Errorf("invalid number of apiKeys: %d", len(apiKeys))
	}

	var proxies *P2BProxies
	if path := os.Getenv("P2B_VPN_PATH"); path != "" {
		fmt.Println(path)

		// Hard-code a couple of proxies
		proxy_uk := P2BProxy{location: "UK", ovpn: "expressvpn_uk.ovpn", port: "54321", url: "http://localhost:54321"}
		err := recycleDockerProxy(path, &proxy_uk)
		if err != nil {
			return nil, err
		}

		proxy_us := P2BProxy{location: "US", ovpn: "expressvpn_us.ovpn", port: "54322", url: "http://localhost:54322"}
		err = recycleDockerProxy(path, &proxy_us)
		if err != nil {
			return nil, err
		}

		proxies = &P2BProxies{proxy: []*P2BProxy{&proxy_uk, &proxy_us}, path: path}
	}

	pbAPIs := make([]*P2BApi, 0)
	for _, apiKey := range apiKeys {
		pbAPIClient := &P2BApi{key: apiKey.Key, secret: apiKey.Secret, proxies: proxies}
		pbAPIs = append(pbAPIs, pbAPIClient)
	}

	return &pbExchange{
		assetConverter: model.P2PB2BAssetConverter,
		apis:           pbAPIs,
		apiNextIndex:   0,
		delimiter:      "_",
		isSimulated:    isSimulated,
	}, nil
}

// nextAPI rotates the API key being used so we can overcome rate limit issues
func (p2b *pbExchange) nextAPI() *P2BApi {
	log.Printf("returning pb API key at index %d", p2b.apiNextIndex)
	api_ := p2b.apis[p2b.apiNextIndex]
	// rotate key for the next call
	p2b.apiNextIndex = (p2b.apiNextIndex + 1) % uint8(len(p2b.apis))
	return api_
}

func (p2b *pbExchange) floatFromString(val string) (float64, error) {
	return strconv.ParseFloat(val, 64)
}

// AddOrder impl.
func (p2b *pbExchange) AddOrder(order *model.Order) (*model.TransactionID, error) {
	market, err := order.Pair.ToString(p2b.assetConverter, p2b.delimiter)
	if err != nil {
		return nil, err
	}

	if p2b.isSimulated {
		log.Printf("not adding order to pb in simulation mode, order=%s\n", *order)
		return model.MakeTransactionID("simulated"), nil
	}

	orderConstraints := p2b.GetOrderConstraints(order.Pair)
	if order.Price.Precision() > orderConstraints.PricePrecision {
		return nil, fmt.Errorf("pb price precision can be a maximum of %d, got %d, value = %.12f", orderConstraints.PricePrecision, order.Price.Precision(), order.Price.AsFloat())
	}
	if order.Volume.Precision() > orderConstraints.VolumePrecision {
		return nil, fmt.Errorf("pb volume precision can be a maximum of %d, got %d, value = %.12f", orderConstraints.VolumePrecision, order.Volume.Precision(), order.Volume.AsFloat())
	}

	resp, err := p2b.nextAPI().createOrder(market, order.Volume.AsString(), order.Price.AsString(), bool(order.OrderAction))
	if err != nil {
		return nil, err
	}

	return model.MakeTransactionID(fmt.Sprintf("%d", resp.Id)), nil
}

// CancelOrder impl.
func (p2b *pbExchange) CancelOrder(txID_ *model.TransactionID, pair model.TradingPair) (model.CancelOrderResult, error) {
	if p2b.isSimulated {
		return model.CancelResultCancelSuccessful, nil
	}

	market, err := pair.ToString(p2b.assetConverter, p2b.delimiter)
	if err != nil {
		return model.CancelResultFailed, err
	}

	txID, err := strconv.ParseUint(txID_.String(), 10, 64)
	if err != nil {
		return model.CancelResultFailed, err
	}

	_, err = p2b.nextAPI().cancelOrder(market, txID)
	if err != nil {
		return model.CancelResultFailed, err
	}

	return model.CancelResultCancelSuccessful, nil
}

// GetAccountBalances impl.
func (p2b *pbExchange) GetAccountBalances(assetList []interface{}) (map[interface{}]model.Number, error) {
	balanceResponse, err := p2b.nextAPI().getAccountBalanaces()
	if err != nil {
		return nil, err
	}

	m := map[interface{}]model.Number{}
	for _, a := range assetList {
		ast, ok := a.(model.Asset)
		if !ok {
			return nil, fmt.Errorf("invalid type of asset passed in, only model.Asset accepted")
		}

		pbAssetString, err := p2b.assetConverter.ToString(ast)
		if err != nil {
			return nil, err
		}
		balItem := balanceResponse[pbAssetString]
		if balItem == nil {
			continue // ignore this asset
		}
		available, err := p2b.floatFromString(balItem.Available)
		if err != nil {
			return nil, err
		}

		freeze, err := p2b.floatFromString(balItem.Freeze)
		if err != nil {
			return nil, err
		}
		m[ast] = *model.NumberFromFloat(available+freeze, precisionBalances)
	}
	return m, nil
}

// GetOrderConstraints impl
func (*pbExchange) GetOrderConstraints(pair *model.TradingPair) *model.OrderConstraints {
	constraints, ok := pbPrecisionMatrix[*pair]
	if !ok {
		log.Printf("pbExchange could not find orderConstraints for trading pair %v, returning nil\n", pair)
		return nil
	}
	return &constraints
}

// GetAssetConverter impl.
func (p2b *pbExchange) GetAssetConverter() *model.AssetConverter {
	return p2b.assetConverter
}

// GetOpenOrders impl.
func (p2b *pbExchange) GetOpenOrders(pairs []*model.TradingPair) (map[model.TradingPair][]model.OpenOrder, error) {
	orders := make(map[model.TradingPair][]model.OpenOrder)
	for _, p := range pairs {
		ordersPair, err := p2b.getOpenOrders(p)
		if err != nil {
			return nil, err
		}
		orders[*p] = ordersPair
	}
	return orders, nil
}

func (p2b *pbExchange) getOpenOrders(pair *model.TradingPair) ([]model.OpenOrder, error) {
	market, err := pair.ToString(p2b.assetConverter, p2b.delimiter)
	if err != nil {
		return nil, err
	}

	orders_, err := p2b.nextAPI().getOpenOrders(market)
	if err != nil {
		return nil, err
	}

	orders := make([]model.OpenOrder, 0)
	if orders_ == nil {
		return orders, nil // return empty list
	}

	for _, o := range *orders_ {
		o := o
		orderConstraints := p2b.GetOrderConstraints(pair)
		order := model.OpenOrder{
			Order: model.Order{
				Pair:        pair,
				OrderAction: model.OrderActionFromString(o.Side),
				OrderType:   model.OrderTypeFromString(o.Type),
				Price:       model.MustNumberFromString(o.Price, orderConstraints.PricePrecision),
				Volume:      model.MustNumberFromString(o.Amount, orderConstraints.VolumePrecision),
				Timestamp:   model.MakeTimestamp(int64(o.Timestamp)),
			},
			ID:             strconv.FormatUint(o.Id, 10),
			StartTime:      model.MakeTimestamp(int64(o.Timestamp)),
			ExpireTime:     model.MakeTimestamp(int64(o.Timestamp) + 2500000),
			VolumeExecuted: model.NumberFromFloat(0.0, orderConstraints.VolumePrecision),
			// @todo hack 0.0 rather than real figure - is this important for our needs?
		}
		orders = append(orders, order)
	}
	return orders, nil
}

// GetOrderBook impl.
func (p2b *pbExchange) GetOrderBook(pair *model.TradingPair, maxCount int32) (*model.OrderBook, error) {
	market, err := pair.ToString(p2b.assetConverter, p2b.delimiter)
	if err != nil {
		return nil, err
	}

	sells_, err := p2b.nextAPI().getOrderBook(market, false, maxCount)
	if err != nil {
		return nil, err
	}
	sells := p2b.readOrders(sells_, pair, model.OrderActionSell)

	buys_, err := p2b.nextAPI().getOrderBook(market, true, maxCount)
	if err != nil {
		return nil, err
	}
	buys := p2b.readOrders(buys_, pair, model.OrderActionBuy)

	ob := model.MakeOrderBook(pair, sells, buys)
	return ob, nil
}

func (p2b *pbExchange) readOrders(orders_ *ExchangeOrders, pair *model.TradingPair, orderAction model.OrderAction) []model.Order {
	orderConstraints := p2b.GetOrderConstraints(pair)
	orders := []model.Order{}
	if orders == nil {
		return orders
	}

	for _, item := range *orders_ {
		orders = append(orders, model.Order{
			Pair:        pair,
			OrderAction: orderAction,
			OrderType:   model.OrderTypeLimit,
			Price:       model.MustNumberFromString(item.Price, orderConstraints.PricePrecision),
			Volume:      model.MustNumberFromString(item.Amount, orderConstraints.VolumePrecision),
			Timestamp:   model.MakeTimestamp(int64(item.Timestamp)),
		})
	}
	return orders
}

// GetTickerPrice impl.
func (p2b *pbExchange) GetTickerPrice(pairs []model.TradingPair) (map[model.TradingPair]api.Ticker, error) {
	priceResult := map[model.TradingPair]api.Ticker{}
	for _, p := range pairs {
		orderConstraints := p2b.GetOrderConstraints(&p)
		market, err := p.ToString(p2b.assetConverter, p2b.delimiter)
		if err != nil {
			return nil, err
		}
		ticker, err := p2b.nextAPI().getTicker(market)
		if err != nil {
			return nil, err
		}

		priceResult[p] = api.Ticker{
			AskPrice: model.MustNumberFromString(ticker.Ask, orderConstraints.PricePrecision),
			BidPrice: model.MustNumberFromString(ticker.Bid, orderConstraints.PricePrecision),
		}
	}

	return priceResult, nil
}

// GetTradeHistory impl.
func (*pbExchange) GetTradeHistory(pair model.TradingPair, maybeCursorStart interface{}, maybeCursorEnd interface{}) (*api.TradeHistoryResult, error) {
	log.Println("pbExchange does not support GetTradeHistory function")
	return nil, ErrorNotSupported
}

// GetLatestTradeCursor impl.
func (*pbExchange) GetLatestTradeCursor() (interface{}, error) {
	timeNowSecs := time.Now().Unix()
	latestTradeCursor := fmt.Sprintf("%d", timeNowSecs)
	return latestTradeCursor, nil
}

// GetTrades impl.
func (*pbExchange) GetTrades(pair *model.TradingPair, maybeCursor interface{}) (*api.TradesResult, error) {
	log.Println("pbExchange does not support GetTrades function")
	return nil, ErrorNotSupported
}

// GetWithdrawInfo impl.
func (*pbExchange) GetWithdrawInfo(
	asset model.Asset,
	amountToWithdraw *model.Number,
	address string,
) (*api.WithdrawInfo, error) {
	log.Println("pbExchange does not support GetWithdrawInfo function")
	return nil, ErrorNotSupported
}

// PrepareDeposit impl.
func (*pbExchange) PrepareDeposit(asset model.Asset, amount *model.Number) (*api.PrepareDepositResult, error) {
	log.Println("pbExchange does not support PrepareDeposit function")
	return nil, ErrorNotSupported
}

// WithdrawFunds impl.
func (*pbExchange) WithdrawFunds(
	asset model.Asset,
	amountToWithdraw *model.Number,
	address string,
) (*api.WithdrawFunds, error) {
	log.Println("pbExchange does not support WithdrawFunds function")
	return nil, ErrorNotSupported
}

// pbPrecisionMatrix describes the price and volume precision and min base volume for each trading pair
// taken from this URL: https://support.pb.com/hc/en-us/articles/360001389366-Price-and-volume-decimal-precision
var pbPrecisionMatrix = map[model.TradingPair]model.OrderConstraints{
	*model.MakeTradingPair(model.SHX, model.USD): *model.MakeOrderConstraints(8, 8, 50.0),
	*model.MakeTradingPair(model.SHX, model.BTC): *model.MakeOrderConstraints(8, 8, 50.0),
	*model.MakeTradingPair(model.SHX, model.ETH): *model.MakeOrderConstraints(8, 8, 50.0),

	//for tests
	*model.MakeTradingPair(model.ETH, model.BTC): *model.MakeOrderConstraints(5, 8, 0.02),
	*model.MakeTradingPair(model.XLM, model.BTC): *model.MakeOrderConstraints(8, 8, 30.0),
}
