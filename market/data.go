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

	// 计算日内系列数据 (3分钟)
	intradayData := calculateIntradaySeries(klines3m)

	// 计算长期数据
	// 计算15分钟系列数据
	midTermData15m := calculateMidTermSeries15m(klines15m)

	// 计算1小时系列数据
	midTermData1h := calculateMidTermSeries1h(klines1h)

	// 计算长期数据 (4小时)
	longerTermData := calculateLongerTermData(klines4h)

	// 使用历史数据接口(当前数据补齐最后一条，所以减去一条历史数据)
	oiData, err := fetchOIData(symbol, "1h", 9)
	if err != nil {
		// OI失败不影响整体,使用默认值
		return nil, fmt.Errorf("获取OI数据失败: %v", err)
	}

	// 计算各种信号
	signalData := make([]*Signal, 24)
	// ATR
	atrData := calculateATRData(klines4h, 14)
	atrSignal := analyzeATRTrend(atrData, 5)
	signalData = append(signalData, &atrSignal)

	// AVL
	avlData := calculateAVL(klines4h)
	avlSignal := analyzeAVLSignal(avlData, 10)
	signalData = append(signalData, &avlSignal)

	// BOLL
	bbData := calculateBollingerBands(klines1h, 20, 2.0)
	bbSignal := analyzeBollingerSignal(bbData, 10)
	signalData = append(signalData, &bbSignal)

	// B.S Vol
	// 获取最新成交数据
	var simulatedTrades []TradeDetail
	simulatedTrades, err = WSMonitorCli.GetCurrentTrades(symbol)
	if err != nil {
		return nil, fmt.Errorf("获取最新成交数据失败: %v", err)
	}
	volumeAnalysis := analyzeTradeFlow(simulatedTrades, 30)
	bsVolSignal := generateBSVolumeSignal(volumeAnalysis, klines3m)
	signalData = append(signalData, &bsVolSignal)

	// CCI
	cciData := calculateCCI(klines4h, 20)
	cciSignal := analyzeCCISignal(cciData, 20)
	signalData = append(signalData, &cciSignal)

	// CMF
	cmfData := calculateCMF(klines4h, 20)
	cmfSignal := analyzeCMFSignal(cmfData)
	signalData = append(signalData, &cmfSignal)

	// DMI
	dmiData := calculateDMI(klines4h, 14)
	dmiSignal := analyzeDMISignal(dmiData, 20)
	signalData = append(signalData, &dmiSignal)

	// EMA20
	emaData := calculateEMAData(klines15m, 20)
	emaSignal := analyzeEMASignal(emaData, 5)
	signalData = append(signalData, &emaSignal)

	// EMV
	emvData := calculateEMV(klines4h, 14, 9)
	emvSignal := analyzeEMVSignal(emvData, 20)
	signalData = append(signalData, &emvSignal)

	// KDJ
	kdjData := calculateKDJ(klines4h, 9, 3, 3)
	kdjSignal := analyzeKDJSignal(kdjData, 20)
	signalData = append(signalData, &kdjSignal)

	// MACD
	macdData := calculateMACDData(klines15m, 12, 26, 9)
	macdSignal := analyzeMACDSignal(macdData, 20)
	signalData = append(signalData, &macdSignal)

	// MFI
	mfiData := calculateMFI(klines4h, 14)
	mfiSignal := analyzeMFISignal(mfiData, 20)
	signalData = append(signalData, &mfiSignal)

	// MTM
	mtmData := calculateMTM(klines4h, 10, 6)
	mtmSignal := analyzeMTMSignal(mtmData, 20)
	signalData = append(signalData, &mtmSignal)

	// OBV
	obvData := CalculateOBV(klines15m)
	obvSignal := analyzeOBVWithPeaks(obvData, 10)
	signalData = append(signalData, &obvSignal)

	// O.I.
	oiSignal := analyzeOISignal(oiData, klines1h, 10)
	signalData = append(signalData, &oiSignal)

	//RSI
	rsiData := calculateRSIData(klines3m, 14)
	rsiSignal := analyzeRSISignal(rsiData, 20)
	signalData = append(signalData, &rsiSignal)

	// SAR
	sarData := calculateSAR(klines4h, 0.02, 0.2, 0.02)
	sarSignal := analyzeSARSignal(sarData, 10)
	signalData = append(signalData, &sarSignal)

	// STOCH RSI
	stochRSIData := calculateStochRSI(klines4h, 14, 14, 3, 3)
	stochRSISignal := analyzeStochRSISignal(stochRSIData, 20)
	signalData = append(signalData, &stochRSISignal)

	//TRIX
	trixData := calculateTRIX(klines4h, 15)
	trixSignal := analyzeTRIXSignal(trixData, 20)
	signalData = append(signalData, &trixSignal)

	// Vol MA
	volumeData := calculateVolumeMA(klines1h, 20)
	volSignal := analyzeVolumeSignal(volumeData, 10)
	signalData = append(signalData, &volSignal)

	// VWAP
	vwapData := calculateVWAP(klines15m, 20)
	vwapSignal := analyzeVWAPSignal(vwapData, 5)
	signalData = append(signalData, &vwapSignal)

	// WMA
	periods := []int{10, 30, 50} // 短期、中期、长期WMA
	multiWMA := calculateMultiPeriodWMA(klines15m, periods)
	// 使用WMA10作为主要分析对象
	wma10Data := multiWMA[10]
	// 分析WMA信号
	wmaSignal := analyzeWMASignal(wma10Data, multiWMA)
	signalData = append(signalData, &wmaSignal)

	// WR
	wrData := calculateWR(klines4h, 14)
	wrSignal := analyzeWRSignal(wrData, 20)
	signalData = append(signalData, &wrSignal)

	// 资金费率
	fundingRate, _ := getFundingRate(symbol)
	if fundingRate > 0 {
		signalData = append(signalData, &Signal{
			Target:     "FundingRate",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "资金费率大于0，开空信号",
		})
	} else {
		signalData = append(signalData, &Signal{
			Target:     "FundingRate",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "资金费率小于0，开多信号",
		})
	}

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		IntradaySeries:    intradayData,
		MidTermSeries15m:  midTermData15m,
		MidTermSeries1h:   midTermData1h,
		LongerTermContext: longerTermData,
		Signals:           signalData,
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
	// data.ATR3 = calculateATR(klines, 3)
	// data.ATR7 = calculateATR(klines, 7)
	// data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	// if len(klines) > 0 {
	// 	data.CurrentVolume = klines[len(klines)-1].Volume
	// 	// 计算平均成交量
	// 	sum := 0.0
	// 	sum20 := 0.0
	// 	count := 0
	// 	for _, k := range klines {
	// 		sum += k.Volume
	// 		if count < 20 {
	// 			sum20 += k.Volume
	// 		}
	// 		count++
	// 	}
	// 	data.AverageVolume = sum / float64(len(klines))
	// 	// 最近20分成交均量
	// 	data.AverageVolume20 = sum20 / 20
	// }

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

	return data
}

// getTakerlongshortRatioData 合约主动买卖量,多空比率
func getTakerlongshortRatioData(symbol string, period string, limit int) float64 {
	api_url := fmt.Sprintf("https://fapi.binance.com/futures/data/takerlongshortRatio?symbol=%s&period=%s&limit=%s", symbol, period, limit)

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
		BuySellRatio string `json:"buySellRatio"`
		BuyVol       string `json:"buyVol"`
		SellVol      string `json:"sellVol"`
		Timestamp    int64  `json:"timestamp"`
	}

	var data []InterestData
	if err := json.Unmarshal(body, &data); err != nil {
		return 0.0
	}

	buySellRatioTotal := 0.0
	for _, item := range data {
		buySellRatio := 0.0
		buySellRatio, _ = strconv.ParseFloat(item.BuySellRatio, 64)
		buySellRatioTotal += buySellRatio
	}

	return buySellRatioTotal / float64(len(data))
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

// getLastPrice 获取最新价格
func GetLastPrice(symbol string) (float64, error) {
	var klines3m []Kline
	klines3m, err := WSMonitorCli.GetCurrentKlines(symbol, "3m") // 多获取一些用于计算
	if err != nil {
		return 0.0, fmt.Errorf("获取3分钟K线失败: %v", err)
	}
	CurrentPrice := klines3m[len(klines3m)-1].Close

	return CurrentPrice, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	sb.WriteString(fmt.Sprintf("current_price = %s, current_ema20 = %.3f, current_macd = %.3f, current_rsi (7 period) = %.3f\n\n",
		priceStr, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	if len(data.Signals) > 0 {
		sb.WriteString("CheckList:\n\n")
		sb.WriteString("| Index| Side | Confidence | Message |\n\n")
		for _, s := range data.Signals {
			sb.WriteString(fmt.Sprintf("| %s | %s | %.1f | %s |\n\n", s.Target, s.Side, s.Confidence, s.Message))
		}
	}

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("- Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("- EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("- MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}
	}

	if data.MidTermSeries15m != nil {
		sb.WriteString("Mid‑term series (15‑minute intervals, oldest → latest):\n\n")

		if len(data.MidTermSeries15m.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("- Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MidPrices)))
		}

		if len(data.MidTermSeries15m.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("- EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.EMA20Values)))
		}

		if len(data.MidTermSeries15m.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("- MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MACDValues)))
		}

		if len(data.MidTermSeries15m.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI7Values)))
		}

		if len(data.MidTermSeries15m.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI14Values)))
		}
	}

	if data.MidTermSeries1h != nil {
		sb.WriteString("Mid‑term series (1‑hour intervals, oldest → latest):\n\n")

		if len(data.MidTermSeries1h.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("- Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MidPrices)))
		}

		if len(data.MidTermSeries1h.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("- EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.EMA20Values)))
		}

		if len(data.MidTermSeries1h.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("- MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MACDValues)))
		}

		if len(data.MidTermSeries1h.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI7Values)))
		}

		if len(data.MidTermSeries1h.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI14Values)))
		}
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

		sb.WriteString(fmt.Sprintf("- 20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("- MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("- RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}
	}

	return sb.String()
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
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

// checkAAboveB 检查A是否在B之上
func CheckAAboveB(dataA, dataB []float64) bool {
	aboveCount := 0
	totalPoints := len(dataA)

	for i := range totalPoints {
		if dataA[i] > dataB[i] {
			aboveCount++
		}
	}

	// 如果A在B之上的点数超过80%，认为A在B之上
	return float64(aboveCount)/float64(totalPoints) > 0.8
}

// LinearRegressionAnalysis 线性回归分析
func LinearRegressionAnalysis(data []float64) (slope, strength float64) {
	n := float64(len(data))
	var sumX, sumY, sumXY, sumXX float64

	for i, price := range data {
		x := float64(i)
		y := price
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// 计算斜率
	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 计算趋势强度 (R-squared)
	meanY := sumY / n
	var totalSS, regSS float64
	for i, price := range data {
		x := float64(i)
		predicted := (slope * x) + (sumY/n - slope*sumX/n)
		totalSS += math.Pow(price-meanY, 2)
		regSS += math.Pow(predicted-meanY, 2)
	}

	rSquared := 0.0
	if totalSS > 0 {
		rSquared = regSS / totalSS
	}

	// 斜率大于0表示上扬，小于0表示下跌，趋势强度使用R-squared
	return slope, rSquared
}

// calculateAvgDistance 计算A和B之间的平均距离
func CalculateAvgDistance(dataA, dataB []float64) float64 {
	var totalDistance float64
	for i := 0; i < len(dataA); i++ {
		totalDistance += dataA[i] - dataB[i]
	}
	return totalDistance / float64(len(dataA))
}

// calculateCorrelation 计算A和B的相关性
func CalculateCorrelation(dataA, dataB []float64) float64 {
	n := float64(len(dataA))

	var sumA, sumB, sumAB, sumA2, sumB2 float64
	for i := range dataA {
		a := dataA[i]
		b := dataB[i]
		sumA += a
		sumB += b
		sumAB += a * b
		sumA2 += a * a
		sumB2 += b * b
	}

	numerator := n*sumAB - sumA*sumB
	denominator := math.Sqrt((n*sumA2 - sumA*sumA) * (n*sumB2 - sumB*sumB))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}
