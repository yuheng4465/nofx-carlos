package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines3m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取3分钟K线数据 (最近10个)
	klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m") // 多获取一些用于计算
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// 获取15分钟K线数据 (最近40个) - 短期趋势
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据 (最近60个) - 中期趋势
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据 (最近60个) - 长期趋势
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h")
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 获取最新成交数据
	var tradeData []Trade
	tradeData, err = WSMonitorCli.GetCurrentTrades(symbol)
	if err != nil {
		return nil, fmt.Errorf("获取最新成交数据失败: %v", err)
	}

	// 计算当前指标 (基于3分钟最新数据)
	currentPrice := klines3m[len(klines3m)-1].Close
	currentEMA20 := calculateEMA(klines3m, 20)
	currentMACD := calculateMACD(klines3m)
	currentRSI7 := calculateRSI(klines3m, 7)

	// 计算价格变化百分比
	// 1小时价格变化 = 20个3分钟K线前的价格
	priceChange1h := 0.0
	if len(klines3m) >= 21 { // 至少需要21根K线 (当前 + 20根前)
		price1hAgo := klines3m[len(klines3m)-21].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}
	// 获取4小时历史OI数据
	oiHistData4h := getOpenInterestHistData(symbol, "4h")

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据 (3分钟)
	intradayData := calculateIntradaySeries(klines3m)

	// 计算长期数据
	// 计算15分钟系列数据
	midTermData15m := calculateMidTermSeries15m(klines15m)

	// 计算1小时系列数据
	midTermData1h := calculateMidTermSeries1h(klines1h)

	// 计算长期数据 (4小时)
	longerTermData := calculateLongerTermData(klines4h)

	// 计算BuySellRatio
	buySellRatio := CalculateBuySellRatio(tradeData)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		MidTermSeries15m:  midTermData15m,
		MidTermSeries1h:   midTermData1h,
		LongerTermContext: longerTermData,
		BuySellRatio:      buySellRatio,
		OiHistData4h:      oiHistData4h,
	}, nil
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算布林带带宽（Bollinger Bandwidth）
	data.BollingerBandwidth = CalculateBollingerBandwidth(klines, 20, 2.0)

	// 成交量加权平均价（VWAP）
	data.VWAPValues = CalculateVWAP(klines)

	// CalculateCMF 计算蔡金资金流
	data.CMFValues = CalculateCMF(klines, 20)

	return data
}

// calculateMidTermSeries15m 计算15分钟系列数据
func calculateMidTermSeries15m(klines []Kline) *MidTermData15m {
	data := &MidTermData15m{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算布林带带宽（Bollinger Bandwidth）
	data.BollingerBandwidth = CalculateBollingerBandwidth(klines, 20, 2.0)

	// 成交量加权平均价（VWAP）
	data.VWAPValues = CalculateVWAP(klines)

	// CalculateCMF 计算蔡金资金流
	data.CMFValues = CalculateCMF(klines, 20)

	return data
}

// calculateMidTermSeries1h 计算1小时系列数据
func calculateMidTermSeries1h(klines []Kline) *MidTermData1h {
	data := &MidTermData1h{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算布林带带宽（Bollinger Bandwidth）
	data.BollingerBandwidth = CalculateBollingerBandwidth(klines, 20, 2.0)

	// 成交量加权平均价（VWAP）
	data.VWAPValues = CalculateVWAP(klines)

	// CalculateCMF 计算蔡金资金流
	data.CMFValues = CalculateCMF(klines, 20)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		sum20 := 0.0
		count := 0
		for _, k := range klines {
			sum += k.Volume
			if count < 20 {
				sum20 += k.Volume
			}
			count++
		}
		data.AverageVolume = sum / float64(len(klines))
		// 最近20分成交均量
		data.AverageVolume20 = sum20 / 20
	}

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算布林带带宽（Bollinger Bandwidth）
	data.BollingerBandwidth = CalculateBollingerBandwidth(klines, 20, 2.0)

	// 成交量加权平均价（VWAP）
	data.VWAPValues = CalculateVWAP(klines)

	// CalculateCMF 计算蔡金资金流
	data.CMFValues = CalculateCMF(klines, 20)

	return data
}

// CalculateBuySellRatio 计算给定交易列表的Buy/Sell Ratio
// 参数 trades: 包含交易方向信息的交易列表
// 返回值 ratio: Buy/Sell Ratio。如果卖出量为0，返回-1表示无穷大（或根据情况返回特殊值）
func CalculateBuySellRatio(trades []Trade) float64 {
	var totalBuyVolume float64
	var totalSellVolume float64

	// 数据过少不计算结果
	if len(trades) < 300 {
		return -1
	}

	for _, trade := range trades {
		if trade.IsTakerSell {
			// 这是主动卖出单
			totalSellVolume += trade.Qty
		} else {
			// 这是主动买入单
			totalBuyVolume += trade.Qty
		}
	}

	// 避免除零错误
	if totalSellVolume == 0 {
		if totalBuyVolume > 0 {
			// 如果卖出量为0但买入量大于0，比率可以视为无穷大，这里返回一个特殊值，例如-1
			return -1
		}
		// 如果买卖量都为0，则比率为0/0，没有意义，返回0
		return 0
	}

	ratio := totalBuyVolume / totalSellVolume
	return ratio
}

// CalculateBollingerBandwidth 计算布林带带宽百分比
func CalculateBollingerBandwidth(klines []Kline, period int, multiplier float64) []float64 {
	// 提取收盘价
	var closes []float64
	for _, k := range klines {
		closes = append(closes, k.Close)
	}

	if len(closes) < 50 {
		return nil
	}

	if len(closes) < period {
		return nil
	}

	bandwidths := make([]float64, len(closes))

	for i := period - 1; i < len(closes); i++ {
		// 计算中轨 (SMA)
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closes[j]
		}
		midBand := sum / float64(period)

		// 计算标准差
		variance := 0.0
		for j := i - period + 1; j <= i; j++ {
			deviation := closes[j] - midBand
			variance += deviation * deviation
		}
		stdDev := math.Sqrt(variance / float64(period))

		// 计算上下轨
		upperBand := midBand + multiplier*stdDev
		lowerBand := midBand - multiplier*stdDev

		// 计算带宽百分比: (上轨 - 下轨) / 中轨 * 100%
		bandwidths[i] = ((upperBand - lowerBand) / midBand) * 100
	}
	return bandwidths
}

// CalculateVWAP 计算成交量加权平均价
func CalculateVWAP(data []Kline) []float64 {
	vwap := make([]float64, len(data))
	cumulativeVolume := 0.0
	cumulativeTypicalPriceVolume := 0.0

	for i, point := range data {
		// 典型价 = (高 + 低 + 收) / 3
		typicalPrice := (point.High + point.Low + point.Close) / 3
		typicalPriceVolume := typicalPrice * point.Volume

		cumulativeTypicalPriceVolume += typicalPriceVolume
		cumulativeVolume += point.Volume

		if cumulativeVolume > 0 {
			vwap[i] = cumulativeTypicalPriceVolume / cumulativeVolume
		}
	}
	return vwap
}

// CalculateCMF 计算蔡金资金流
func CalculateCMF(data []Kline, period int) []float64 {
	cmf := make([]float64, len(data))

	for i := period - 1; i < len(data); i++ {
		sumMoneyFlowVolume := 0.0
		sumVolume := 0.0

		for j := i - period + 1; j <= i; j++ {
			point := data[j]
			// 资金流乘数 = [(收盘价 - 最低价) - (最高价 - 收盘价)] / (最高价 - 最低价)
			moneyFlowMultiplier := ((point.Close - point.Low) - (point.High - point.Close)) / (point.High - point.Low)
			moneyFlowVolume := moneyFlowMultiplier * point.Volume

			sumMoneyFlowVolume += moneyFlowVolume
			sumVolume += point.Volume
		}

		if sumVolume > 0 {
			cmf[i] = sumMoneyFlowVolume / sumVolume
		}
	}
	return cmf
}

// CalculateOBV 计算能量潮
func CalculateOBV(klines []Kline, period int) ([]float64, error) {
	// 提取收盘价
	// 提取收盘价和成交量
	var closes, volumes []float64
	for _, point := range klines {
		closes = append(closes, point.Close)
		volumes = append(volumes, point.Volume)
	}

	if len(closes) < 50 {
		return nil, fmt.Errorf("数据少于50条")
	}

	if len(closes) < period {
		return nil, fmt.Errorf("数据点数不足")
	}

	obv := make([]float64, len(closes))
	obv[0] = volumes[0] // 初始OBV为第一天的成交量

	for i := 1; i < len(closes); i++ {
		if closes[i] > closes[i-1] {
			// 价格上涨，成交量加入OBV
			obv[i] = obv[i-1] + volumes[i]
		} else if closes[i] < closes[i-1] {
			// 价格下跌，成交量从OBV中减去
			obv[i] = obv[i-1] - volumes[i]
		} else {
			// 价格持平，OBV不变
			obv[i] = obv[i-1]
		}
	}
	return obv, nil
}

// getOpenInterestData 获取OI数据（获取当前未平仓合约数）
func getOpenInterestData(symbol string) (*OIData, error) {
	api_url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 创建 HTTP 客户端并使用自定义 Transport
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(api_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getOpenInterestHistData 获取OI数据（合约持仓量历史）
func getOpenInterestHistData(symbol string, period string) float64 {
	api_url := fmt.Sprintf("https://fapi.binance.com/futures/data/openInterestHist?symbol=%s&period=%s", symbol, period)

	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 创建 HTTP 客户端并使用自定义 Transport
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(api_url)
	if err != nil {
		return 0.0
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.0
	}

	type InterestData struct {
		SumOpenInterest string `json:"sumOpenInterest"`
		Symbol          string `json:"symbol"`
		Timestamp       int64  `json:"timestamp"`
	}

	var data []InterestData
	if err := json.Unmarshal(body, &data); err != nil {
		return 0.0
	}

	lastItem := data[len(data)-1]
	sumOpenInterest, _ := strconv.ParseFloat(lastItem.SumOpenInterest, 64)

	return sumOpenInterest
}

// getFundingRate 获取资金费率
func getFundingRate(symbol string) (float64, error) {
	api_url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 创建 HTTP 客户端并使用自定义 Transport
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(api_url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("current_price = %.2f, current_ema20 = %.3f, current_macd = %.3f, current_rsi (7 period) = %.3f\n\n",
		data.CurrentPrice, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil && data.OiHistData4h > 0 {
		sb.WriteString(fmt.Sprintf("OI: %.2f vs. OI4h: %.2f\n\n",
			data.OpenInterest.Latest, data.OiHistData4h))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.8f\n\n", data.FundingRate))

	if data.BuySellRatio >= 0 {
		sb.WriteString(fmt.Sprintf("BuySellRatio: %.2f\n\n", data.BuySellRatio))
	}

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}

		if len(data.IntradaySeries.BollingerBandwidth) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger Bandwidth (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.BollingerBandwidth)))
		}

		if len(data.IntradaySeries.VWAPValues) > 0 {
			sb.WriteString(fmt.Sprintf("VWAP : %s\n\n", formatFloatSlice(data.IntradaySeries.VWAPValues)))
		}

		if len(data.IntradaySeries.CMFValues) > 0 {
			sb.WriteString(fmt.Sprintf("CMF & OBV (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.CMFValues)))
		}
	}

	if data.MidTermSeries15m != nil {
		sb.WriteString("Mid‑term series (15‑minute intervals, oldest → latest):\n\n")

		if len(data.MidTermSeries15m.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MidPrices)))
		}

		if len(data.MidTermSeries15m.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.EMA20Values)))
		}

		if len(data.MidTermSeries15m.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MACDValues)))
		}

		if len(data.MidTermSeries15m.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI7Values)))
		}

		if len(data.MidTermSeries15m.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI14Values)))
		}

		if len(data.MidTermSeries15m.BollingerBandwidth) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger Bandwidth (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.BollingerBandwidth)))
		}

		if len(data.MidTermSeries15m.VWAPValues) > 0 {
			sb.WriteString(fmt.Sprintf("VWAP : %s\n\n", formatFloatSlice(data.MidTermSeries15m.VWAPValues)))
		}

		if len(data.MidTermSeries15m.CMFValues) > 0 {
			sb.WriteString(fmt.Sprintf("CMF & OBV (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.CMFValues)))
		}
	}

	if data.MidTermSeries1h != nil {
		sb.WriteString("Mid‑term series (1‑hour intervals, oldest → latest):\n\n")

		if len(data.MidTermSeries1h.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MidPrices)))
		}

		if len(data.MidTermSeries1h.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.EMA20Values)))
		}

		if len(data.MidTermSeries1h.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MACDValues)))
		}

		if len(data.MidTermSeries1h.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI7Values)))
		}

		if len(data.MidTermSeries1h.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI14Values)))
		}

		if len(data.MidTermSeries1h.BollingerBandwidth) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger Bandwidth (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.BollingerBandwidth)))
		}

		if len(data.MidTermSeries1h.VWAPValues) > 0 {
			sb.WriteString(fmt.Sprintf("VWAP : %s\n\n", formatFloatSlice(data.MidTermSeries1h.VWAPValues)))
		}

		if len(data.MidTermSeries1h.CMFValues) > 0 {
			sb.WriteString(fmt.Sprintf("CMF & OBV (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.CMFValues)))
		}
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

		sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f vs. 14‑Period ATR: %.3f\n\n",
			data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f vs. Volume MA(20): %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume, data.LongerTermContext.AverageVolume20))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}

		if len(data.LongerTermContext.BollingerBandwidth) > 0 {
			sb.WriteString(fmt.Sprintf("Bollinger Bandwidth (20‑period): %s\n\n", formatFloatSlice(data.LongerTermContext.BollingerBandwidth)))
		}

		if len(data.LongerTermContext.VWAPValues) > 0 {
			sb.WriteString(fmt.Sprintf("VWAP : %s\n\n", formatFloatSlice(data.LongerTermContext.VWAPValues)))
		}

		if len(data.LongerTermContext.CMFValues) > 0 {
			sb.WriteString(fmt.Sprintf("CMF & OBV (20‑period): %s\n\n", formatFloatSlice(data.LongerTermContext.CMFValues)))
		}
	}

	return sb.String()
}

// formatFloatSlice 格式化float64切片为字符串
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.3f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
