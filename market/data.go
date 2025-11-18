package market

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
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

	// 获取1d K线数据 (最近60个) - 长期趋势
	// klines1d, err = WSMonitorCli.GetCurrentKlines(symbol, "1d")
	// if err != nil {
	// 	return nil, fmt.Errorf("获取1天K线失败: %v", err)
	// }

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
	intradayData := calculateIntradaySeries(klines3m, symbol)

	// 计算长期数据
	// 计算15分钟系列数据
	midTermData15m := calculateMidTermSeries15m(klines15m, symbol)

	// 计算1小时系列数据
	midTermData1h := calculateMidTermSeries1h(klines1h, symbol)

	// 计算长期数据 (4小时)
	longerTermData := calculateLongerTermData(klines4h, symbol)

	// 计算长期数据（1天）
	// longerTermData1d := calculateLongerTermData1d(klines1d, symbol)

	// 获取最新成交数据
	var simulatedTrades []TradeDetail
	simulatedTrades, err = WSMonitorCli.GetCurrentTrades(symbol)
	if err != nil {
		return nil, fmt.Errorf("获取最新成交数据失败: %v", err)
	}
	volumeAnalysis := analyzeTradeFlow(simulatedTrades, 30)
	intradayData.BSVOL = volumeAnalysis

	// 资金费率
	apiClient := NewAPIClient()
	fundingRate, _ := apiClient.getFundingRate(symbol)
	fundingRate = fundingRate * 100
	intradayData.FundingRate = fundingRate

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      midTermData1h.OI,
		IntradaySeries:    intradayData,
		MidTermSeries15m:  midTermData15m,
		MidTermSeries1h:   midTermData1h,
		LongerTermContext: longerTermData,
		// longerTermData1d:  longerTermData1d,
		// Signal:     signal,
		// SignalList: signalList,
	}, nil
}

// 获取策略判断所需数据key
func getStrategyKeys() []string {
	return []string{
		"Price_3m", "Price_15m", "Price_1h", "Price_4h",
		"BOLLMid_3m", "BOLLUp_3m", "BOLLLow_3m",
		"BOLLMid_15m", "BOLLUp_15m", "BOLLLow_15m",
		"BOLLMid_1h", "BOLLUp_1h", "BOLLLow_1h",
		"BOLLMid_4h", "BOLLUp_4h", "BOLLLow_4h",
		"EMA20_3m", "EMA20_15m", "EMA20_1h", "EMA20_4h",
		"MACD_3m", "MACD_15m", "MACD_1h", "MACD_4h",
		"MACDSignal_3m", "MACDSignal_15m", "MACDSignal_1h", "MACDSignal_4h",
		"MACDHist_3m", "MACDHist_15m", "MACDHist_1h", "MACDHist_4h",
		"RSI_3m", "RSI_15m", "RSI_1h", "RSI_4h",
		"OBV_3m", "OBV_15m", "OBV_1h", "OBV_4h",
		"CCI_3m", "CCI_15m", "CCI_1h", "CCI_4h",
		"WR_3m", "WR_15m", "WR_1h", "WR_4h",
		"ATR_3m", "ATR_15m", "ATR_1h", "ATR_4h",
		"CMF_3m", "CMF_15m", "CMF_1h", "CMF_4h",
		"VOL_3m", "VOL_15m", "VOL_1h", "VOL_4h",
		"FundingRate", "OI", "SellVol", "BuyVol",
	}
}

// ComprehensiveAnalysis 综合决策引擎
func ComprehensiveAnalysis(engine *DecisionEngine, data *Data) (*Signal, error) {
	// 获取策略判断所需数据
	strategyData := getStrategyData(data)
	requiredKeys := getStrategyKeys()

	// 验证数据是否完整
	for _, key := range requiredKeys {
		value, exists := strategyData.Metrics[key]
		if !exists {
			return nil, fmt.Errorf("决策配置数据: %v不存在", key)
		}

		// 检查是否为 []float64 且长度大于 0
		if slice, ok := value.([]float64); ok {
			if len(slice) == 0 {
				return nil, fmt.Errorf("决策配置数据: %v为空", key)
			}
		} else {
			return nil, fmt.Errorf("决策配置数据: %v类型错误", key)
		}
	}

	// 执行策略
	matched, err := engine.ExecuteStrategy(strategyData)
	if matched.Matched {
		return &Signal{
			SignalType:   "none",
			Side:         matched.MatchedStrategy.Side,
			Confidence:   matched.MatchedStrategy.Confidence,
			Strategy:     &matched.MatchedStrategy,
			StrategyData: strategyData,
			Message:      matched.MatchedStrategy.Description,
		}, nil
	} else {
		return &Signal{
			SignalType:   "none",
			Side:         SideNone,
			Confidence:   0,
			Strategy:     &matched.MatchedStrategy,
			StrategyData: strategyData,
			Message:      err.Error(),
		}, nil
	}
}

// 泛型字段提取函数
func Map[T any, U any](slice []T, mapper func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}

// 获取策略所用指标数据
func getStrategyData(data *Data) *StrategyData {

	MACD_4h := Map(data.LongerTermContext.MACD, func(item *MACDData) float64 {
		return item.MACDLine
	})
	macdSignal_3m := Map(data.IntradaySeries.MACD, func(item *MACDData) float64 {
		return item.SignalLine
	})
	macdSignal_15m := Map(data.MidTermSeries15m.MACD, func(item *MACDData) float64 {
		return item.SignalLine
	})
	macdSignal_1h := Map(data.MidTermSeries1h.MACD, func(item *MACDData) float64 {
		return item.SignalLine
	})
	macdSignal_4h := Map(data.LongerTermContext.MACD, func(item *MACDData) float64 {
		return item.SignalLine
	})

	macdHist_3m := Map(data.IntradaySeries.MACD, func(item *MACDData) float64 {
		return item.Histogram
	})
	macdHist_15m := Map(data.MidTermSeries15m.MACD, func(item *MACDData) float64 {
		return item.Histogram
	})
	macdHist_1h := Map(data.MidTermSeries1h.MACD, func(item *MACDData) float64 {
		return item.Histogram
	})
	macdHist_4h := Map(data.LongerTermContext.MACD, func(item *MACDData) float64 {
		return item.Histogram
	})

	// 布林带
	BOLLMid_3m := Map(data.IntradaySeries.BOLL, func(item *BollingerBandData) float64 {
		return item.MiddleBand
	})
	BOLLUp_3m := Map(data.IntradaySeries.BOLL, func(item *BollingerBandData) float64 {
		return item.UpperBand
	})
	BOLLLow_3m := Map(data.IntradaySeries.BOLL, func(item *BollingerBandData) float64 {
		return item.LowerBand
	})

	BOLLMid_15m := Map(data.MidTermSeries15m.BOLL, func(item *BollingerBandData) float64 {
		return item.MiddleBand
	})
	BOLLUp_15m := Map(data.MidTermSeries15m.BOLL, func(item *BollingerBandData) float64 {
		return item.UpperBand
	})
	BOLLLow_15m := Map(data.MidTermSeries15m.BOLL, func(item *BollingerBandData) float64 {
		return item.LowerBand
	})

	BOLLMid_1h := Map(data.MidTermSeries1h.BOLL, func(item *BollingerBandData) float64 {
		return item.MiddleBand
	})
	BOLLUp_1h := Map(data.MidTermSeries1h.BOLL, func(item *BollingerBandData) float64 {
		return item.UpperBand
	})
	BOLLLow_1h := Map(data.MidTermSeries1h.BOLL, func(item *BollingerBandData) float64 {
		return item.LowerBand
	})

	BOLLMid_4h := Map(data.LongerTermContext.BOLL, func(item *BollingerBandData) float64 {
		return item.MiddleBand
	})
	BOLLUp_4h := Map(data.LongerTermContext.BOLL, func(item *BollingerBandData) float64 {
		return item.UpperBand
	})
	BOLLLow_4h := Map(data.LongerTermContext.BOLL, func(item *BollingerBandData) float64 {
		return item.LowerBand
	})

	// RSI
	RSI_4h := Map(data.LongerTermContext.RSI, func(item *RSIData) float64 {
		return item.RSI
	})

	// OBV
	OBV_3m := Map(data.IntradaySeries.OBV, func(item *OBVData) float64 {
		return item.OBV
	})
	OBV_15m := Map(data.MidTermSeries15m.OBV, func(item *OBVData) float64 {
		return item.OBV
	})
	OBV_1h := Map(data.MidTermSeries1h.OBV, func(item *OBVData) float64 {
		return item.OBV
	})
	OBV_4h := Map(data.LongerTermContext.OBV, func(item *OBVData) float64 {
		return item.OBV
	})

	// CCI
	CCI_3m := Map(data.IntradaySeries.CCI, func(item *CCIData) float64 {
		return item.CCI
	})
	CCI_15m := Map(data.MidTermSeries15m.CCI, func(item *CCIData) float64 {
		return item.CCI
	})
	CCI_1h := Map(data.MidTermSeries1h.CCI, func(item *CCIData) float64 {
		return item.CCI
	})
	CCI_4h := Map(data.LongerTermContext.CCI, func(item *CCIData) float64 {
		return item.CCI
	})

	// CMF
	CMF_3m := Map(data.IntradaySeries.CMF, func(item *CMFData) float64 {
		return item.CMF
	})
	CMF_15m := Map(data.MidTermSeries15m.CMF, func(item *CMFData) float64 {
		return item.CMF
	})
	CMF_1h := Map(data.MidTermSeries1h.CMF, func(item *CMFData) float64 {
		return item.CMF
	})
	CMF_4h := Map(data.LongerTermContext.CMF, func(item *CMFData) float64 {
		return item.CMF
	})

	// WR
	WR_3m := Map(data.IntradaySeries.WR, func(item *WRData) float64 {
		return item.WR
	})
	WR_15m := Map(data.MidTermSeries15m.WR, func(item *WRData) float64 {
		return item.WR
	})
	WR_1h := Map(data.MidTermSeries15m.WR, func(item *WRData) float64 {
		return item.WR
	})
	WR_4h := Map(data.LongerTermContext.WR, func(item *WRData) float64 {
		return item.WR
	})
	OI := []float64{}
	if len(data.IntradaySeries.OI) > 0 {
		OI = getOIList(data.IntradaySeries.OI, 10)
	}

	var FundingRate []float64
	FundingRate = append(FundingRate, data.IntradaySeries.FundingRate)

	SellVol := Map(data.IntradaySeries.BSVOL, func(item *VolumeAnalysis) float64 {
		return item.ActiveSellVolume
	})
	BuyVol := Map(data.IntradaySeries.BSVOL, func(item *VolumeAnalysis) float64 {
		return item.ActiveBuyVolume
	})

	ATR_4h := Map(data.LongerTermContext.ATR, func(item *ATRData) float64 {
		return item.ATR
	})

	strategyData := &StrategyData{}
	strategyData.Name = "StrategyData"
	strategyData.Metrics = map[string]interface{}{
		"Price_3m":       data.IntradaySeries.MidPrices,
		"Price_15m":      data.MidTermSeries15m.MidPrices,
		"Price_1h":       data.MidTermSeries1h.MidPrices,
		"Price_4h":       data.LongerTermContext.MidPrices,
		"BOLLMid_3m":     BOLLMid_3m,
		"BOLLUp_3m":      BOLLUp_3m,
		"BOLLLow_3m":     BOLLLow_3m,
		"BOLLMid_15m":    BOLLMid_15m,
		"BOLLUp_15m":     BOLLUp_15m,
		"BOLLLow_15m":    BOLLLow_15m,
		"BOLLMid_1h":     BOLLMid_1h,
		"BOLLUp_1h":      BOLLUp_1h,
		"BOLLLow_1h":     BOLLLow_1h,
		"BOLLMid_4h":     BOLLMid_4h,
		"BOLLUp_4h":      BOLLUp_4h,
		"BOLLLow_4h":     BOLLLow_4h,
		"EMA20_3m":       data.IntradaySeries.EMA20Values,
		"EMA20_15m":      data.MidTermSeries15m.EMA20Values,
		"EMA20_1h":       data.MidTermSeries1h.EMA20Values,
		"EMA20_4h":       data.LongerTermContext.EMA20,
		"MACD_3m":        data.IntradaySeries.MACDValues,
		"MACD_15m":       data.MidTermSeries15m.MACDValues,
		"MACD_1h":        data.MidTermSeries1h.MACDValues,
		"MACD_4h":        MACD_4h,
		"MACDSignal_3m":  macdSignal_3m,
		"MACDSignal_15m": macdSignal_15m,
		"MACDSignal_1h":  macdSignal_1h,
		"MACDSignal_4h":  macdSignal_4h,
		"MACDHist_3m":    macdHist_3m,
		"MACDHist_15m":   macdHist_15m,
		"MACDHist_1h":    macdHist_1h,
		"MACDHist_4h":    macdHist_4h,
		"RSI_3m":         data.IntradaySeries.RSI14Values,
		"RSI_15m":        data.MidTermSeries15m.RSI14Values,
		"RSI_1h":         data.MidTermSeries1h.RSI14Values,
		"RSI_4h":         RSI_4h,
		"OBV_3m":         OBV_3m,
		"OBV_15m":        OBV_15m,
		"OBV_1h":         OBV_1h,
		"OBV_4h":         OBV_4h,
		"CCI_3m":         CCI_3m,
		"CCI_15m":        CCI_15m,
		"CCI_1h":         CCI_1h,
		"CCI_4h":         CCI_4h,
		"WR_3m":          WR_3m,
		"WR_15m":         WR_15m,
		"WR_1h":          WR_1h,
		"WR_4h":          WR_4h,
		"ATR_3m":         data.IntradaySeries.ATR,
		"ATR_15m":        data.MidTermSeries15m.ATR,
		"ATR_1h":         data.MidTermSeries1h.ATR,
		"ATR_4h":         ATR_4h,
		"CMF_3m":         CMF_3m,
		"CMF_15m":        CMF_15m,
		"CMF_1h":         CMF_1h,
		"CMF_4h":         CMF_4h,
		"VOL_3m":         data.IntradaySeries.Volume,
		"VOL_15m":        data.MidTermSeries15m.Volume,
		"VOL_1h":         data.MidTermSeries1h.Volume,
		"VOL_4h":         data.LongerTermContext.Volume,
		"FundingRate":    FundingRate,
		"OI":             OI,
		"SellVol":        SellVol,
		"BuyVol":         BuyVol,
	}

	return strategyData
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
func calculateIntradaySeries(klines []Kline, symbol string) *IntradayData {
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
		data.Volume = append(data.Volume, klines[i].Volume)

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

	// 计算3m ATR14
	data.ATR = calculateATRList(klines, 14)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.CCI = calculateCCI(klines, 20)
	data.CMF = calculateCMF(klines, 20)
	simulatedTrades, _ := WSMonitorCli.GetCurrentTrades(symbol)
	data.BSVOL = analyzeTradeFlow(simulatedTrades, 30)
	data.WR = calculateWR(klines, 14)
	OI, err := fetchOIData(symbol, "5m", 10)
	if err != nil {
		log.Println("Error fetching OI data: %w", err)
	}
	if len(OI) > 0 {
		data.OI = OI
	}
	data.OBV = calculateOBV(klines)

	return data
}

// calculateMidTermSeries15m 计算15分钟系列数据
func calculateMidTermSeries15m(klines []Kline, symbol string) *MidTermData15m {
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
		data.Volume = append(data.Volume, klines[i].Volume)

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

	// 15m指标
	data.ATR = calculateATRList(klines, 14)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.CCI = calculateCCI(klines, 20)
	data.CMF = calculateCMF(klines, 20)
	// data.OI, _ = fetchOIData(symbol, "15m", 20)
	data.EMA = calculateEMAData(klines, 20)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.RSI = calculateRSIData(klines, 14)
	data.OBV = calculateOBV(klines)
	// data.VWAP = calculateVWAP(klines, 20)
	data.WR = calculateWR(klines, 14)
	data.VOL = calculateVolumeMA(klines, 20)

	return data
}

// calculateMidTermSeries1h 计算1小时系列数据
func calculateMidTermSeries1h(klines []Kline, symbol string) *MidTermData1h {
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
		data.Volume = append(data.Volume, klines[i].Volume)

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

	// 1h指标
	data.ATR = calculateATRList(klines, 14)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.CCI = calculateCCI(klines, 20)
	data.CMF = calculateCMF(klines, 20)
	data.EMA = calculateEMAData(klines, 20)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.OBV = calculateOBV(klines)
	// data.OI, _ = fetchOIData(symbol, "1h", 20)
	data.RSI = calculateRSIData(klines, 14)
	// data.TRIX = calculateTRIX(klines, 15)
	data.VOL = calculateVolumeMA(klines, 20)
	data.WR = calculateWR(klines, 14)
	// vwapData1h := calculateVWAP(klines1h, 20)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline, symbol string) *LongerTermData {
	data := &LongerTermData{}

	// 计算EMA
	// data.EMA20 = calculateEMA(klines, 20)
	// data.EMA50 = calculateEMA(klines, 50)

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
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20 = append(data.EMA20, ema20)
		}
	}

	// for i := start; i < len(klines); i++ {
	// 	if i >= 25 {
	// 		macd := calculateMACD(klines[:i+1])
	// 		data.MACDValues = append(data.MACDValues, macd)
	// 	}
	// 	if i >= 14 {
	// 		rsi14 := calculateRSI(klines[:i+1], 14)
	// 		data.RSI14Values = append(data.RSI14Values, rsi14)
	// 	}
	// }

	// 4h指标
	data.ATR = calculateATRData(klines, 14)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.CCI = calculateCCI(klines, 20)
	data.CMF = calculateCMF(klines, 20)
	// data.DMI = calculateDMI(klines, 14)
	data.EMA = calculateEMAData(klines, 20)
	// data.EMV = calculateEMV(klines, 14, 9)
	// data.KDJ = calculateKDJ(klines, 9, 3, 3)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	// data.MFI = calculateMFI(klines, 14)
	// data.MTM = calculateMTM(klines, 10, 6)
	// data.OI, _ = fetchOIData(symbol, "4h", 20)
	data.OBV = calculateOBV(klines)
	data.RSI = calculateRSIData(klines, 14)
	// data.SAR = calculateSAR(klines, 0.02, 0.2, 0.02)
	// data.StochRSI = calculateStochRSI(klines, 14, 14, 3, 3)
	// data.TRIX = calculateTRIX(klines, 15)
	data.WR = calculateWR(klines, 14)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData1d(klines []Kline, symbol string) *LongerTermData1d {
	data := &LongerTermData1d{}

	start := len(klines) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
	}

	// 1d指标
	data.AVL = calculateAVL(klines)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.CMF = calculateCMF(klines, 20)
	data.EMA = calculateEMAData(klines, 20)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.OBV = calculateOBV(klines)
	// data.OI, _ = fetchOIData(symbol, "1d", 20)
	data.RSI = calculateRSIData(klines, 14)
	data.TRIX = calculateTRIX(klines, 15)
	data.VOL = calculateVolumeMA(klines, 20)

	return data
}

// getLastPrice 获取最新价格
func GetLastPrice(symbol string) (float64, error) {
	var klines3m []Kline
	klines3m, err := WSMonitorCli.GetCurrentKlines(symbol, "3m")
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
	sb.WriteString(fmt.Sprintf("当前: Price = %s, EMA20 = %.3f, MACD = %.3f, RSI7 = %.3f\n\n",
		priceStr, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	sb.WriteString("指标数据: \n\n")
	sb.WriteString("| Index | Data |\n")
	sb.WriteString("| ----- | ----- |\n")
	requiredKeys := getStrategyKeys()
	for _, key := range requiredKeys {
		value, exists := data.Signal.StrategyData.Metrics[key]
		if !exists {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", key, formatStringStruct(value)))
	}
	sb.WriteString("| -----| ----- | ----- |\n\n")

	return sb.String()
}

func formatStringStruct[T any](slice T) string {
	jsonData, err := json.Marshal(slice)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return ""
	}

	// 将字节切片转换为字符串
	jsonString := string(jsonData)

	return jsonString
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

// isStaleData detects stale data (consecutive price freeze)
// Fix DOGEUSDT-style issue: consecutive N periods with completely unchanged prices indicate data source anomaly
func isStaleData(klines []Kline, symbol string) bool {
	if len(klines) < 5 {
		return false // Insufficient data to determine
	}

	// Detection threshold: 5 consecutive 3-minute periods with unchanged price (15 minutes without fluctuation)
	const stalePriceThreshold = 5
	const priceTolerancePct = 0.0001 // 0.01% fluctuation tolerance (avoid false positives)

	// Take the last stalePriceThreshold K-lines
	recentKlines := klines[len(klines)-stalePriceThreshold:]
	firstPrice := recentKlines[0].Close

	// Check if all prices are within tolerance
	for i := 1; i < len(recentKlines); i++ {
		priceDiff := math.Abs(recentKlines[i].Close-firstPrice) / firstPrice
		if priceDiff > priceTolerancePct {
			return false // Price fluctuation exists, data is normal
		}
	}

	// Additional check: MACD and volume
	// If price is unchanged but MACD/volume shows normal fluctuation, it might be a real market situation (extremely low volatility)
	// Check if volume is also 0 (data completely frozen)
	allVolumeZero := true
	for _, k := range recentKlines {
		if k.Volume > 0 {
			allVolumeZero = false
			break
		}
	}

	if allVolumeZero {
		log.Printf("⚠️  %s stale data confirmed: price freeze + zero volume", symbol)
		return true
	}

	// Price frozen but has volume: might be extremely low volatility market, allow but log warning
	log.Printf("⚠️  %s detected extreme price stability (no fluctuation for %d consecutive periods), but volume is normal", symbol, stalePriceThreshold)
	return false
}
