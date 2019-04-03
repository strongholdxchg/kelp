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
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	p2bBaseURL    = "https://p2pb2b.io"
	p2bApiPrefix  = "/api/v1"
	proxyAttempts = 4
	startSleep    = 5.0
	multSleep     = 1.5
)

// Common structs
type P2BProxy struct {
	location, ovpn, port string
}

type P2BProxies struct {
	proxy []*P2BProxy
	index uint64
	path  string
}

type P2BApi struct {
	key     string
	secret  string
	proxies *P2BProxies
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
	Total  int `json:"total,omitempty"`
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
	Orders  *ExchangeOrders `json:"orders"`
	Records *ExchangeOrders `json:"records"`
}

type GetOrdersRequest struct {
	*P2BRequest
	P2BLimitOffset
	Market string `json:"market" binding:"required"`
}

type GetOrdersResponse struct {
	P2BResponse
	Result *GetOrdersResult `json:"result"`
}

type ExchangeOrder struct {
	Id        uint64  `json:"orderId" binding:"required"`
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

type ExchangeOrders []*ExchangeOrder

type ActionOrderResult ExchangeOrder

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
type TickerPriceResult struct {
	Bid  string `json:"bid" binding:"required"`
	Ask  string `json:"ask" binding:"required"`
	Low  string `json:"low" binding:"required"`
	High string `json:"high" binding:"required"`
	Last string `json:"last" binding:"required"`
	Vol  string `json:"vol" binding:"required"`
}

type TickerPriceResponse struct {
	P2BResponse
	Result *TickerPriceResult `json:"result"`
}

func recycleDockerProxy(path string, proxy *P2BProxy) error {
	script := filepath.Join(path, "openvpn.sh")
	// fmt.Println(script, proxy)
	ovpn := filepath.Join(path, proxy.ovpn)
	command := exec.Command("bash", script, proxy.location, ovpn, proxy.port)
	// fmt.Println(proxy.location, ovpn, proxy.port)
	return command.Run()
}

//
func (p2b *P2BApi) unsuccessful() error {
	return errors.New("UNSUCCESSFUL_REQUEST")
}

func (p2b *P2BApi) getAccountBalanaces() (GetAccountBalancesResult, error) {
	p2bRequest := P2BRequest{
		Request: fmt.Sprintf("%s/account/balances", p2bApiPrefix),
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

func (p2b *P2BApi) getOpenOrders(market string) (*ExchangeOrders, error) {
	p2bRequest := P2BRequest{
		Request: fmt.Sprintf("%s/orders", p2bApiPrefix),
	}
	offset := 0
	limit := 100
	orders := make(ExchangeOrders, 0)
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
		//fmt.Println(*response.Result, response.Result.Records)
		if response.Result.Records != nil && response.Result.Total > 0 {
			orders = append(orders, *response.Result.Records...)
		}
		if response.Result.Total < limit {
			return &orders, nil
		}
		offset += limit
	}
}

func (p2b *P2BApi) createOrder(market, amount, price string, sell bool) (*ActionOrderResult, error) {
	p2bRequest := P2BRequest{
		Request: fmt.Sprintf("%s/order/new", p2bApiPrefix),
	}

	side := "buy"
	if sell {
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
		Request: fmt.Sprintf("%s/order/cancel", p2bApiPrefix),
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

func (p2b *P2BApi) getTicker(market string) (*TickerPriceResult, error) {
	var response TickerPriceResponse
	paras := map[string]string{
		"market": market,
	}
	request := fmt.Sprintf("%s/public/ticker", p2bApiPrefix)
	err := p2b.get(request, &response, paras)
	if err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, p2b.unsuccessful()
	}
	return response.Result, nil
}

func (p2b *P2BApi) getOrderBook(market string, sell bool, maxCount int32) (*ExchangeOrders, error) {
	offset := 0
	limit := 100
	orders := make(ExchangeOrders, 0)

	side := "buy"
	if sell {
		side = "sell"
	}

	for {
		if offset >= int(maxCount) {
			break
		}
		var response GetOrdersResponse
		paras := map[string]string{
			"market": market,
			"side":   side,
			"limit":  strconv.FormatUint(uint64(limit), 10),
			"offset": strconv.FormatUint(uint64(offset), 10),
		}
		request := fmt.Sprintf("%s/public/book", p2bApiPrefix)
		err := p2b.get(request, &response, paras)
		if err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, p2b.unsuccessful()
		}
		if response.Result.Orders != nil && response.Result.Total > 0 {
			orders = append(orders, *response.Result.Orders...)
		}
		if response.Result.Total < limit {
			break
		}
		offset += limit
	}
	result := orders[0:maxCount]
	return &result, nil
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
	urlPrefix := p2bBaseURL
	if p2b.proxies != nil {
		urlPrefix = fmt.Sprintf("http://localhost:%s", p2b.proxies.proxy[p2b.proxies.index].port)
	}
	url := fmt.Sprintf("%s%s", urlPrefix, p2bR.Request)
	//fmt.Println(data, hex_, sig)

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
	urlPrefix := p2bBaseURL
	if p2b.proxies != nil {
		urlPrefix = fmt.Sprintf("http://localhost:%s", p2b.proxies.proxy[p2b.proxies.index].port)
	}
	url := fmt.Sprintf("%s%s", urlPrefix, request_)
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
	fmt.Println(request.URL.String())

	attempts := 1
	sleep := startSleep
	if p2b.proxies != nil {
		attempts = proxyAttempts
	}
	var err error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			err = recycleDockerProxy(p2b.proxies.path, p2b.proxies.proxy[p2b.proxies.index])
			if err != nil {
				return err
			}
		}

		err = p2b.submit_(request, response)
		if err == nil {
			break // from for
		}
		time.Sleep(time.Duration(sleep) * time.Second)
		fmt.Print(".")
		sleep *= multSleep
	}
	if p2b.proxies != nil {
		p2b.proxies.index = (p2b.proxies.index + 1) % uint64(len(p2b.proxies.proxy))
	}
	return err
}

func (p2b *P2BApi) submit_(request *http.Request, response interface{}) error {
	if p2b.proxies != nil {
		fmt.Printf("TRYING proxy: %s\n", p2b.proxies.proxy[p2b.proxies.index].location)
	}

	client := &http.Client{}
	httpResponse, err := client.Do(request)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	fmt.Println("response Status Code:", httpResponse.StatusCode)

	if httpResponse.StatusCode >= 200 && httpResponse.StatusCode < 300 {
		body, _ := ioutil.ReadAll(httpResponse.Body)
		bodyString := string(body)
		//fmt.Println("response Body:", bodyString)
		bodyReader := strings.NewReader(bodyString)
		decoder := json.NewDecoder(bodyReader)
		return decoder.Decode(&response)
	}
	return errors.New(fmt.Sprintf("BAD_STATUS_CODE: %d", httpResponse.StatusCode))
}

//func main() {
//	api := P2BApi{
//		key:    "b97cd1a2d30a83ba8417994117c5c78d", //"95733fc9a4971707fba0a8c215f57740"
//		secret: "f8b1e34e77c3d048efe6b2a1117f1646", //"6f5cb056eb57f269dc6e15caf1dc6c08"
//	}
//	res, err := api.getAccountBalanaces()
//	res, err := api.getTicker("ETH_BTC")
//	res, err := api.createOrder("XLM_BTC", "10", "0.00003", true)
//	res, err := api.getOpenOrders("XLM_BTC")
//	res, err := api.getOrderBook("XLM_BTC", true)
//	res, err := api.cancelOrder("XLM_BTC", 10802264)
//
//	fmt.Println(res, err)
//	for _, o := range *res {
//		fmt.Println(*o)
//	}
//}
