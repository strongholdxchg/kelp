package p2pb2b

import (
	"errors"
	"fmt"
	"github.com/stellar/kelp/api"
	"github.com/stellar/kelp/model"
	"log"
	"math"
)

// ensure that pbExchange conforms to the Exchange interface
var _ api.Exchange = &pbExchange{}
var ErrorNotSupported = errors.New("FUNCTION_NOT_SUPPORTED")

// const precisionBalances = 10

// pbExchange is the implementation for the p2pb2b Exchange
type pbExchange struct {
	// assetConverter           *model.AssetConverter
	// assetConverterOpenOrders *model.AssetConverter // pb uses different symbols when fetching open orders!
	// apis                     []*pbapi.pbApi
	// apiNextIndex             uint8
	// delimiter    string
	// withdrawKeys asset2Address2Key
	// isSimulated  bool // will simulate add and cancel orders if this is true
}

//type asset2Address2Key map[model.Asset]map[string]string
//
//func (m asset2Address2Key) getKey(asset model.Asset, address string) (string, error) {
//	address2Key, ok := m[asset]
//	if !ok {
//		return "", fmt.Errorf("asset (%v) is not registered in asset2Address2Key: %v", asset, m)
//	}
//
//	key, ok := address2Key[address]
//	if !ok {
//		return "", fmt.Errorf("address is not registered in asset2Address2Key: %v (asset = %v)", address, asset)
//	}
//
//	return key, nil
//}
//
// makepbExchange is a factory method to make the pb exchange
func makepbExchange(apiKeys []api.ExchangeAPIKey, isSimulated bool) (api.Exchange, error) {
	if len(apiKeys) == 0 || len(apiKeys) > math.MaxUint8 {
		return nil, fmt.Errorf("invalid number of apiKeys: %d", len(apiKeys))
	}

	pbAPIs := []*pbapi.pbApi{}
	for _, apiKey := range apiKeys {
		pbAPIClient := pbapi.New(apiKey.Key, apiKey.Secret)
		pbAPIs = append(pbAPIs, pbAPIClient)
	}

	return &pbExchange{
		assetConverter:           model.pbAssetConverter,
		assetConverterOpenOrders: model.pbAssetConverterOpenOrders,
		apis:                     pbAPIs,
		apiNextIndex:             0,
		delimiter:                "",
		withdrawKeys:             asset2Address2Key{},
		isSimulated:              isSimulated,
	}, nil
}

// nextAPI rotates the API key being used so we can overcome rate limit issues
func (k *pbExchange) nextAPI() *pbapi.pbApi {
	log.Printf("returning pb API key at index %d", k.apiNextIndex)
	api := k.apis[k.apiNextIndex]
	// rotate key for the next call
	k.apiNextIndex = (k.apiNextIndex + 1) % uint8(len(k.apis))
	return api
}

// AddOrder impl.
func (b2b *pbExchange) AddOrder(order *model.Order) (*model.TransactionID, error) {
	//pairStr, e := order.Pair.ToString(k.assetConverter, k.delimiter)
	//if e != nil {
	//	return nil, e
	//}
	//
	//if k.isSimulated {
	//	log.Printf("not adding order to pb in simulation mode, order=%s\n", *order)
	//	return model.MakeTransactionID("simulated"), nil
	//}
	//
	//orderConstraints := k.GetOrderConstraints(order.Pair)
	//if order.Price.Precision() > orderConstraints.PricePrecision {
	//	return nil, fmt.Errorf("pb price precision can be a maximum of %d, got %d, value = %.12f", orderConstraints.PricePrecision, order.Price.Precision(), order.Price.AsFloat())
	//}
	//if order.Volume.Precision() > orderConstraints.VolumePrecision {
	//	return nil, fmt.Errorf("pb volume precision can be a maximum of %d, got %d, value = %.12f", orderConstraints.VolumePrecision, order.Volume.Precision(), order.Volume.AsFloat())
	//}
	//
	//args := map[string]string{
	//	"price": order.Price.AsString(),
	//}
	//resp, e := k.nextAPI().AddOrder(
	//	pairStr,
	//	order.OrderAction.String(),
	//	order.OrderType.String(),
	//	order.Volume.AsString(),
	//	args,
	//)
	//if e != nil {
	//	return nil, e
	//}
	//
	//// expected case for production orders
	//if len(resp.TransactionIds) == 1 {
	//	return model.MakeTransactionID(resp.TransactionIds[0]), nil
	//}
	//
	//if len(resp.TransactionIds) > 1 {
	//	return nil, fmt.Errorf("there was more than 1 transctionId: %s", resp.TransactionIds)
	//}
	//
	return nil, fmt.Errorf("no transactionIds returned from order creation")
}

// CancelOrder impl.
func (k *pbExchange) CancelOrder(txID *model.TransactionID, pair model.TradingPair) (model.CancelOrderResult, error) {
	//if k.isSimulated {
	//	return model.CancelResultCancelSuccessful, nil
	//}
	//
	//// we don't actually use the pair for pb
	//resp, e := k.nextAPI().CancelOrder(txID.String())
	//if e != nil {
	//	return model.CancelResultFailed, e
	//}
	//
	//if resp.Count > 1 {
	//	log.Printf("warning: count from a cancelled order is greater than 1: %d\n", resp.Count)
	//}
	//
	//// TODO 2 - need to figure out whether count = 0 could also mean that it is pending cancellation
	//if resp.Count == 0 {
	//	return model.CancelResultFailed, nil
	//}
	//// resp.Count == 1 here
	//
	//if resp.Pending {
	//	return model.CancelResultPending, nil
	//}
	return model.CancelResultCancelSuccessful, nil
}

// GetAccountBalances impl.
func (k *pbExchange) GetAccountBalances(assetList []model.Asset) (map[model.Asset]model.Number, error) {
	//balanceResponse, e := k.nextAPI().Balance()
	//if e != nil {
	//	return nil, e
	//}
	//
	//m := map[model.Asset]model.Number{}
	//for _, a := range assetList {
	//	pbAssetString, e := k.assetConverter.ToString(a)
	//	if e != nil {
	//		// discard partially built map for now
	//		return nil, e
	//	}
	//	bal := getFieldValue(*balanceResponse, pbAssetString)
	//	m[a] = *model.NumberFromFloat(bal, precisionBalances)
	//}
	//return m, nil
	return nil, nil
}

//func getFieldValue(object pbapi.BalanceResponse, fieldName string) float64 {
//	r := reflect.ValueOf(object)
//	f := reflect.Indirect(r).FieldByName(fieldName)
//	return f.Interface().(float64)
//}

// GetOrderConstraints impl
func (k *pbExchange) GetOrderConstraints(pair *model.TradingPair) *model.OrderConstraints {
	constraints, ok := pbPrecisionMatrix[*pair]
	if !ok {
		log.Printf("pbExchange could not find orderConstraints for trading pair %v, returning nil\n", pair)
		return nil
	}
	return &constraints
}

// GetAssetConverter impl.
func (k *pbExchange) GetAssetConverter() *model.AssetConverter {
	// return k.assetConverter
	return nil
}

// GetOpenOrders impl.
func (k *pbExchange) GetOpenOrders(pairs []*model.TradingPair) (map[model.TradingPair][]model.OpenOrder, error) {
	//openOrdersResponse, e := k.nextAPI().OpenOrders(map[string]string{})
	//if e != nil {
	//	return nil, e
	//}
	//
	//// convert to a map so we can easily search for the existence of a trading pair
	//// pb uses different symbols when fetching open orders!
	//pairsMap, e := model.TradingPairs2Strings2(k.assetConverterOpenOrders, "", pairs)
	//if e != nil {
	//	return nil, e
	//}
	//
	//m := map[model.TradingPair][]model.OpenOrder{}
	//for ID, o := range openOrdersResponse.Open {
	//	// pb uses different symbols when fetching open orders!
	//	pair, e := model.TradingPairFromString(3, k.assetConverterOpenOrders, o.Description.AssetPair)
	//	if e != nil {
	//		return nil, e
	//	}
	//
	//	if _, ok := pairsMap[*pair]; !ok {
	//		// skip open orders for pairs that were not requested
	//		continue
	//	}
	//
	//	if _, ok := m[*pair]; !ok {
	//		m[*pair] = []model.OpenOrder{}
	//	}
	//	if _, ok := m[model.TradingPair{Base: pair.Quote, Quote: pair.Base}]; ok {
	//		return nil, fmt.Errorf("open orders are listed with repeated base/quote pairs for %s", *pair)
	//	}
	//
	//	orderConstraints := k.GetOrderConstraints(pair)
	//	m[*pair] = append(m[*pair], model.OpenOrder{
	//		Order: model.Order{
	//			Pair:        pair,
	//			OrderAction: model.OrderActionFromString(o.Description.Type),
	//			OrderType:   model.OrderTypeFromString(o.Description.OrderType),
	//			Price:       model.MustNumberFromString(o.Description.PrimaryPrice, orderConstraints.PricePrecision),
	//			Volume:      model.MustNumberFromString(o.Volume, orderConstraints.VolumePrecision),
	//			Timestamp:   model.MakeTimestamp(int64(o.OpenTime)),
	//		},
	//		ID:             ID,
	//		StartTime:      model.MakeTimestamp(int64(o.StartTime)),
	//		ExpireTime:     model.MakeTimestamp(int64(o.ExpireTime)),
	//		VolumeExecuted: model.NumberFromFloat(o.VolumeExecuted, orderConstraints.VolumePrecision),
	//	})
	//}
	//return m, nil
	return nil, nil
}

// GetOrderBook impl.
func (k *pbExchange) GetOrderBook(pair *model.TradingPair, maxCount int32) (*model.OrderBook, error) {
	//pairStr, e := pair.ToString(k.assetConverter, k.delimiter)
	//if e != nil {
	//	return nil, e
	//}
	//
	//pbob, e := k.nextAPI().Depth(pairStr, int(maxCount))
	//if e != nil {
	//	return nil, e
	//}
	//
	//asks := k.readOrders(pbob.Asks, pair, model.OrderActionSell)
	//bids := k.readOrders(pbob.Bids, pair, model.OrderActionBuy)
	//ob := model.MakeOrderBook(pair, asks, bids)
	//return ob, nil
	return nil, nil
}

//func (k *pbExchange) readOrders(obi []pbapi.OrderBookItem, pair *model.TradingPair, orderAction model.OrderAction) []model.Order {
//	orderConstraints := k.GetOrderConstraints(pair)
//	orders := []model.Order{}
//	for _, item := range obi {
//		orders = append(orders, model.Order{
//			Pair:        pair,
//			OrderAction: orderAction,
//			OrderType:   model.OrderTypeLimit,
//			Price:       model.NumberFromFloat(item.Price, orderConstraints.PricePrecision),
//			Volume:      model.NumberFromFloat(item.Amount, orderConstraints.VolumePrecision),
//			Timestamp:   model.MakeTimestamp(item.Ts),
//		})
//	}
//	return orders
//}

// GetTickerPrice impl.
func (k *pbExchange) GetTickerPrice(pairs []model.TradingPair) (map[model.TradingPair]api.Ticker, error) {
	//pairsMap, e := model.TradingPairs2Strings(k.assetConverter, k.delimiter, pairs)
	//if e != nil {
	//	return nil, e
	//}
	//
	//resp, e := k.nextAPI().Ticker(values(pairsMap)...)
	//if e != nil {
	//	return nil, e
	//}
	//
	//priceResult := map[model.TradingPair]api.Ticker{}
	//for _, p := range pairs {
	//	orderConstraints := k.GetOrderConstraints(&p)
	//	pairTickerInfo := resp.GetPairTickerInfo(pairsMap[p])
	//	priceResult[p] = api.Ticker{
	//		AskPrice: model.MustNumberFromString(pairTickerInfo.Ask[0], orderConstraints.PricePrecision),
	//		BidPrice: model.MustNumberFromString(pairTickerInfo.Bid[0], orderConstraints.PricePrecision),
	//	}
	//}
	//
	//return priceResult, nil
	return nil, nil
}

//// values gives you the values of a map
//// TODO 2 - move to autogenerated generic function
//func values(m map[model.TradingPair]string) []string {
//	values := []string{}
//	for _, v := range m {
//		values = append(values, v)
//	}
//	return values
//}

// GetTradeHistory impl.
func (k *pbExchange) GetTradeHistory(pair model.TradingPair, maybeCursorStart interface{}, maybeCursorEnd interface{}) (*api.TradeHistoryResult, error) {
	log.Println("pbExchange does not support GetTradeHistory function")
	return nil, ErrorNotSupported
}

// GetTrades impl.
func (k *pbExchange) GetTrades(pair *model.TradingPair, maybeCursor interface{}) (*api.TradesResult, error) {
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
}
