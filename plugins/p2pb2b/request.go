package p2pb2b

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Common structs
type P2BApi struct {
	key    string
	secret string
}

type P2BRequest struct {
	Request string `json:"request" binding:"required"`
	Nonce   int64  `json:"nonce" binding:"required"`
}

type P2BResponse struct {
	Success bool        `json:"success" binding:"required"`
	Message interface{} `json:"message"`
}

type P2BLimitOffset struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Total  int `json:"limit,omitempty"`
}

// Account balance structs
type GetAccountBalancesResult map[string]*GetAccountBalanceItem

type GetAccountBalancesRequest struct {
	*P2BRequest
}

type GetAccountBalancesResponse struct {
	P2BResponse
	Result GetAccountBalancesResult `json:"result"`
}

type GetAccountBalanceItem struct {
	Available string `json:"available"`
	Freeze    string `json:"freeze"`
}

// Order structs
type GetOrdersResult struct {
	P2BLimitOffset
	Orders *Orders `json:"orders"`
	Result *Orders `json:"result"`
}

type GetOrdersRequest struct {
	*P2BRequest
	P2BLimitOffset
	Market string `json:"market" binding:"required"`
}

type GetOrdersResponse struct {
	P2BResponse
	Result GetOrdersResult `json:"result"`
}

type Order struct {
	Id        uint64  `json:"id" binding:"required"`
	Left      string  `json:"left"`
	Market    string  `json:"market"`
	Amount    string  `json:"amount"`
	Type      string  `json:"type"`
	Price     string  `json:"price"`
	Timestamp float64 `json:"timestamp"`
	Side      string  `json:"side"`
	DealFee   string  `json:"dealFee"`
	TakerFee  string  `json:"takerFee"`
	MakerFee  string  `json:"makerFee"`
	DealStock string  `json:"dealStock"`
	DealMoney string  `json:"dealMoney"`
}

type Orders []*Order

type ActionOrderResult Order

type ActionOrderResponse struct {
	P2BResponse
	Result *ActionOrderResult `json:"result"`
}

// CreateOrder structs
type CreateOrderRequest struct {
	*P2BRequest
	Market string `json:"market" binding:"required"`
	Amount string `json:"amount" binding:"required"`
	Side   string `json:"side" binding:"required"`
	Price  string `json:"price" binding:"required"`
}

// CancelOrder structs
type CancelOrderRequest struct {
	*P2BRequest
	Market string `json:"market" binding:"required"`
	Id     uint64 `json:"orderId" binding:"required"`
}

// TickerPrice structs
type TickerPricesResult map[string]*TickerPriceItem

type TickerPricesResponse struct {
	P2BResponse
	Result TickerPricesResult `json:"result"`
}

type TickerPriceItem struct {
	At     uint64             `json:"at" binding:"required"`
	Ticker *TickerPriceTicker `json:"ticker" binding:"required"`
}

type TickerPriceTicker struct {
	Bid  string `json:"bid" binding:"required"`
	Ask  string `json:"ask" binding:"required"`
	Low  string `json:"low" binding:"required"`
	High string `json:"high" binding:"required"`
	Last string `json:"last" binding:"required"`
	Vol  string `json:"vol" binding:"required"`
}

//
func (p2b *P2BApi) unsuccessful() error {
	return errors.New("UNSUCCESSFUL_REQUEST")
}

func (p2b *P2BApi) getAccountBalanaces() (GetAccountBalancesResult, error) {
	p2bRequest := P2BRequest{
		Request: "/api/v1/account/balances",
	}
	request := GetAccountBalancesRequest{
		P2BRequest: &p2bRequest,
	}

	var response GetAccountBalancesResponse
	err := p2b.post(&p2bRequest, &request, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, p2b.unsuccessful()
	}
	return response.Result, nil
}

func (p2b *P2BApi) getOpenOrders(market string) (*Orders, error) {
	p2bRequest := P2BRequest{
		Request: "/api/v1/orders",
	}
	offset := 0
	limit := 100
	orders := make(Orders, 0)
	for {
		request := GetOrdersRequest{
			P2BRequest: &p2bRequest,
			Market:     market,
			P2BLimitOffset: P2BLimitOffset{
				Offset: offset,
				Limit:  limit,
			},
		}

		var response GetOrdersResponse
		err := p2b.post(&p2bRequest, &request, &response)
		if err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, p2b.unsuccessful()
		}
		orders = append(orders, *response.Result.Result...)
		if response.Result.Total < limit {
			return &orders, nil
		}
		offset += limit
	}
}

func (p2b *P2BApi) createOrder(market, amount, price string, buy bool) (*ActionOrderResult, error) {
	p2bRequest := P2BRequest{
		Request: "/api/v1/order/new",
	}

	var side string
	if buy {
		side = "buy"
	} else {
		side = "sell"
	}

	request := CreateOrderRequest{
		P2BRequest: &p2bRequest,
		Market:     market,
		Amount:     amount,
		Price:      price,
		Side:       side,
	}

	var response ActionOrderResponse
	err := p2b.post(&p2bRequest, &request, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, p2b.unsuccessful()
	}
	return response.Result, nil
}

func (p2b *P2BApi) cancelOrder(market string, id uint64) (*ActionOrderResult, error) {
	p2bRequest := P2BRequest{
		Request: "/api/v1/order/cancel",
	}

	request := CancelOrderRequest{
		P2BRequest: &p2bRequest,
		Market:     market,
		Id:         id,
	}

	var response ActionOrderResponse
	err := p2b.post(&p2bRequest, &request, &response)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, p2b.unsuccessful()
	}
	return response.Result, nil
}

func (p2b *P2BApi) getTicker(market string) (TickerPricesResult, error) {
	var response TickerPricesResponse
	paras := map[string]string{
		"market": market,
	}
	err := p2b.get("/api/v1/public/ticker", &response, paras)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, p2b.unsuccessful()
	}
	return response.Result, nil
}

func (p2b *P2BApi) getOrderBook(market string, buy bool) (*Orders, error) {
	offset := 0
	limit := 100
	orders := make(Orders, 0)

	var side string
	if buy {
		side = "buy"
	} else {
		side = "sell"
	}

	for {
		var response GetOrdersResponse
		paras := map[string]string{
			"market": market,
			"side":   side,
			"limit":  strconv.FormatUint(uint64(limit), 10),
			"offset": strconv.FormatUint(uint64(offset), 10),
		}
		err := p2b.get("/api/v1/public/book", &response, paras)
		if err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, p2b.unsuccessful()
		}
		if response.Result.Orders != nil {
			orders = append(orders, *response.Result.Orders...)
		}
		if response.Result.Total < limit {
			return &orders, nil
		}
		offset += limit
	}
}

func (p2b *P2BApi) post(p2bR *P2BRequest, request_, response interface{}) error {
	p2bR.Nonce = time.Now().UTC().Unix()

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(request_)
	data := strings.TrimSpace(string(b.String()))
	hex_ := base64.StdEncoding.EncodeToString([]byte(data))
	h := hmac.New(sha512.New, []byte(p2b.secret))
	h.Write([]byte(hex_))
	sig := hex.EncodeToString(h.Sum(nil))
	url := fmt.Sprintf("https://p2pb2b.io%s", p2bR.Request)

	request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return err
	}
	request.Header.Set("X-TXC-APIKEY", p2b.key)
	request.Header.Set("X-TXC-PAYLOAD", hex_)
	request.Header.Set("X-TXC-SIGNATURE", sig)
	request.Header.Set("Content-Type", "application/json")

	return p2b.submit(request, response)
}

func (p2b *P2BApi) get(request_ string, response interface{}, paras map[string]string) error {
	url := fmt.Sprintf("https://p2pb2b.io%s", request_)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	query := request.URL.Query()
	if paras != nil {
		for k, v := range paras {
			query.Add(k, v)
		}
	}
	request.URL.RawQuery = query.Encode()
	return p2b.submit(request, response)
}

func (p2b *P2BApi) submit(request *http.Request, response interface{}) error {
	client := &http.Client{}
	httpResponse, err := client.Do(request)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 200 && httpResponse.StatusCode < 300 {
		decoder := json.NewDecoder(httpResponse.Body)
		err = decoder.Decode(&response)
		return err
	}

	return errors.New(fmt.Sprintf("BAD_STATUS_CODE: %d", httpResponse.StatusCode))
}

//func main() {
//	api := P2BApi{
//		key:    "b97cd1a2d30a83ba8417994117c5c78d", //"95733fc9a4971707fba0a8c215f57740"
//		secret: "f8b1e34e77c3d048efe6b2a1117f1646", //"6f5cb056eb57f269dc6e15caf1dc6c08"
//	}
//	//res, err := api.createOrder("XLM_BTC", "10", "0.00003", false)
//	//res, err := api.cancelOrder("XLM_BTC", 10523428)
//	//res, err := api.getAccountBalanaces()
//	res, err := api.getOpenOrders("XLM_BTC")
//	//res, err := api.getTicker("ETH_BTC")
//	//res, err := api.getOrderBook("XLM_BTC", true)
//	fmt.Println(res, err)
//	//for _, o := range *res {
//	//	fmt.Println(*o)
//	//}
//}
