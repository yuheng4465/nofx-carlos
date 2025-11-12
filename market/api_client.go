package market

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL = "https://fapi.binance.com"
	// baseURL = "https://demo-fapi.binance.com"
)

type APIClient struct {
	client *http.Client
}

func NewAPIClient() *APIClient {
	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	return &APIClient{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// getOpenInterestData 获取OI数据（获取当前未平仓合约数）
func (c *APIClient) getOpenInterestData(symbol string) (*OIData, error) {
	api_url := fmt.Sprintf("%s/fapi/v1/openInterest?symbol=%s", baseURL, symbol)

	resp, err := c.client.Get(api_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Symbol:       result.Symbol,
		OpenInterest: oi,
		Timestamp:    result.Time,
	}, nil
}

// openInterestHist 获取OI趋势分析（合约持仓量历史）
func (c *APIClient) getOpenInterestHist(symbol string, period string, limit int) ([]*OIData, error) {
	api_url := fmt.Sprintf("%s/futures/data/openInterestHist?symbol=%s&period=%s&limit=%d", baseURL, symbol, period, limit)

	resp, err := c.client.Get(api_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type InterestData struct {
		SumOpenInterest string `json:"sumOpenInterest"`
		Symbol          string `json:"symbol"`
		Timestamp       int64  `json:"timestamp"`
	}

	var data []InterestData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	history := make([]*OIData, len(data))
	for _, item := range data {
		sumOpenInterest, _ := strconv.ParseFloat(item.SumOpenInterest, 64)
		history = append(history, &OIData{
			Symbol:       symbol,
			OpenInterest: sumOpenInterest,
			Timestamp:    item.Timestamp,
		})
	}

	return history, nil
}

// getFundingRate 获取资金费率
func (c *APIClient) getFundingRate(symbol string) (float64, error) {
	api_url := fmt.Sprintf("%s/fapi/v1/premiumIndex?symbol=%s", baseURL, symbol)

	resp, err := c.client.Get(api_url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)
	return rate, nil
}

func (c *APIClient) GetExchangeInfo() (*ExchangeInfo, error) {
	url := fmt.Sprintf("%s/fapi/v1/exchangeInfo", baseURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var exchangeInfo ExchangeInfo
	err = json.Unmarshal(body, &exchangeInfo)
	if err != nil {
		return nil, err
	}

	return &exchangeInfo, nil
}

// 获取近期成交
func (c *APIClient) GetTrades(symbol string, limit int) ([]TradeDetail, error) {
	url := fmt.Sprintf("%s/fapi/v1/aggTrades", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tradeInfo []TradeInfo
	err = json.Unmarshal(body, &tradeInfo)
	if err != nil {
		return nil, err
	}

	var trades []TradeDetail
	for _, tr := range tradeInfo {
		trade, err := parseTrade(tr)
		if err != nil {
			log.Printf("解析成交数据失败: %v", err)
			continue
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

func parseTrade(tr TradeInfo) (TradeDetail, error) {
	var trade TradeDetail

	// 设置直接匹配的字段
	trade.TradeID = tr.TradeID
	trade.Timestamp = tr.Timestamp
	trade.IsBuyerMaker = tr.IsBuyerMaker

	// 处理Price字段的转换
	trade.Price, _ = strconv.ParseFloat(tr.Price, 64)
	trade.Quantity, _ = strconv.ParseFloat(tr.Quantity, 64)

	return trade, nil
}

func (c *APIClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("interval", interval)
	q.Add("limit", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var klineResponses []KlineResponse
	err = json.Unmarshal(body, &klineResponses)
	if err != nil {
		return nil, err
	}

	var klines []Kline
	for _, kr := range klineResponses {
		kline, err := parseKline(kr)
		if err != nil {
			log.Printf("解析K线数据失败: %v", err)
			continue
		}
		klines = append(klines, kline)
	}

	return klines, nil
}

func parseKline(kr KlineResponse) (Kline, error) {
	var kline Kline

	if len(kr) < 11 {
		return kline, fmt.Errorf("invalid kline data")
	}

	// 解析各个字段
	kline.OpenTime = int64(kr[0].(float64))
	kline.Open, _ = strconv.ParseFloat(kr[1].(string), 64)
	kline.High, _ = strconv.ParseFloat(kr[2].(string), 64)
	kline.Low, _ = strconv.ParseFloat(kr[3].(string), 64)
	kline.Close, _ = strconv.ParseFloat(kr[4].(string), 64)
	kline.Volume, _ = strconv.ParseFloat(kr[5].(string), 64)
	kline.CloseTime = int64(kr[6].(float64))
	kline.QuoteVolume, _ = strconv.ParseFloat(kr[7].(string), 64)
	kline.Trades = int(kr[8].(float64))
	kline.TakerBuyBaseVolume, _ = strconv.ParseFloat(kr[9].(string), 64)
	kline.TakerBuyQuoteVolume, _ = strconv.ParseFloat(kr[10].(string), 64)

	return kline, nil
}

func (c *APIClient) GetCurrentPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/fapi/v1/ticker/price", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("symbol", symbol)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var ticker PriceTicker
	err = json.Unmarshal(body, &ticker)
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}
