package market

// RSIData 存储RSI数据点
type RSIData struct {
	OpenTime   int64
	ClosePrice float64
	RSI        float64
}

// RSISignal 存储RSI分析信号
type RSISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateRSI 计算RSI指标
func calculateRSIData(klines []Kline, period int) []*RSIData {
	var rsiList []*RSIData
	var gains []float64
	var losses []float64

	for i, kline := range klines {
		closePrice := kline.Close

		if i > 0 {
			prevClose := klines[i-1].Close
			change := closePrice - prevClose

			if change > 0 {
				gains = append(gains, change)
				losses = append(losses, 0.0)
			} else {
				gains = append(gains, 0.0)
				losses = append(losses, -change) // 存储为正值
			}
		} else {
			// 第一根K线，变化为0
			gains = append(gains, 0.0)
			losses = append(losses, 0.0)
		}

		// 计算RSI
		if i >= period {
			avgGain := 0.0
			avgLoss := 0.0

			// 计算初始平均值
			for j := 0; j < period; j++ {
				avgGain += gains[i-j]
				avgLoss += losses[i-j]
			}
			avgGain /= float64(period)
			avgLoss /= float64(period)

			// 计算RS
			rs := 0.0
			if avgLoss != 0 {
				rs = avgGain / avgLoss
			}

			// 计算RSI
			rsi := 100.0 - (100.0 / (1.0 + rs))

			rsiList = append(rsiList, &RSIData{
				OpenTime:   kline.OpenTime,
				ClosePrice: closePrice,
				RSI:        rsi,
			})
		}
	}
	return rsiList
}

// analyzeRSISignal 分析RSI数据，生成交易信号
func analyzeRSISignal(rsiData []*RSIData, lookback int) Signal {
	if len(rsiData) < lookback+1 {
		return Signal{Target: "RSI", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := rsiData[len(rsiData)-1]
	prev := rsiData[len(rsiData)-2]

	// 信号1: 超买超卖线穿越
	// 从下方上穿30线 (超卖反弹)
	if prev.RSI < 30 && current.RSI >= 30 {
		return Signal{
			Target:     "RSI",
			SignalType: "bullish_oversold",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "RSI从超卖区反弹！潜在开多信号",
		}
	}
	// 从上下穿70线 (超买回落)
	if prev.RSI > 70 && current.RSI <= 70 {
		return Signal{
			Target:     "RSI",
			SignalType: "bearish_overbought",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "RSI从超买区回落！潜在开空信号",
		}
	}

	// 信号2: 中轴线穿越
	// 在强势区域上穿50
	if current.RSI > 40 && current.RSI < 80 {
		if prev.RSI < 50 && current.RSI >= 50 {
			return Signal{
				Target:     "RSI",
				SignalType: "bullish_momentum",
				Side:       "buy",
				Confidence: 0.6,
				Message:    "RSI在强势区上穿50，上涨动量增强",
			}
		}
	}

	// 在弱势区域下穿50
	if current.RSI > 20 && current.RSI < 60 {
		if prev.RSI > 50 && current.RSI <= 50 {
			return Signal{
				Target:     "RSI",
				SignalType: "bearish_momentum",
				Side:       "sell",
				Confidence: 0.6,
				Message:    "RSI在弱势区下穿50，下跌动量增强",
			}
		}
	}

	// 信号3: 背离检测
	bullishDivergence := detectBullishDivergence(rsiData, lookback)
	bearishDivergence := detectBearishDivergence(rsiData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "RSI",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现RSI底背离！强烈开多信号！",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "RSI",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现RSI顶背离！强烈开空信号！",
		}
	}

	// 信号4: RSI水平判断
	if current.RSI > 70 {
		return Signal{
			Target:     "RSI",
			SignalType: "overbought",
			Side:       "sell",
			Confidence: 0.5,
			Message:    "RSI处于超买区，警惕回调，不宜追多",
		}
	}

	if current.RSI < 30 {
		return Signal{
			Target:     "RSI",
			SignalType: "oversold",
			Side:       "buy",
			Confidence: 0.5,
			Message:    "RSI处于超卖区，警惕反弹，不宜追空",
		}
	}

	return Signal{Target: "RSI", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确信号"}
}

// detectBullishDivergence 检测看多背离 (价格新低，RSI更高低点)
func detectBullishDivergence(rsiData []*RSIData, lookback int) bool {
	if len(rsiData) < lookback {
		return false
	}

	// 寻找价格低点和RSI低点
	recentPriceLow, recentPriceLowIndex := findPriceLowRSI(rsiData, lookback)
	recentRSILow, _ := findRSILow(rsiData, lookback)

	// 寻找前一个价格低点和RSI低点
	prevLookback := recentPriceLowIndex
	if prevLookback < 5 {
		return false
	}
	prevPriceLow, _ := findPriceLowRSI(rsiData[:prevLookback], 5)
	prevRSILow, _ := findRSILow(rsiData[:prevLookback], 5)

	// 检查背离：价格创新低，但RSI低点抬高
	if recentPriceLow < prevPriceLow && recentRSILow > prevRSILow {
		return true
	}

	return false
}

// detectBearishDivergence 检测看空背离 (价格新高，RSI更低高点)
func detectBearishDivergence(rsiData []*RSIData, lookback int) bool {
	if len(rsiData) < lookback {
		return false
	}

	// 寻找价格高点和RSI高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighRSI(rsiData, lookback)
	recentRSIHigh, _ := findRSIHigh(rsiData, lookback)

	// 寻找前一个价格高点和RSI高点
	prevLookback := recentPriceHighIndex
	if prevLookback < 5 {
		return false
	}
	prevPriceHigh, _ := findPriceHighRSI(rsiData[:prevLookback], 5)
	prevRSIHigh, _ := findRSIHigh(rsiData[:prevLookback], 5)

	// 检查背离：价格创新高，但RSI高点降低
	if recentPriceHigh > prevPriceHigh && recentRSIHigh < prevRSIHigh {
		return true
	}

	return false
}

// 辅助函数：在RSI数据中寻找价格低点
func findPriceHighRSI(data []*RSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].ClosePrice
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].ClosePrice > high {
			high = data[i].ClosePrice
			index = i
		}
	}
	return high, index
}

// 辅助函数：在RSI数据中寻找价格低点
func findPriceLowRSI(data []*RSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].ClosePrice
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].ClosePrice < low {
			low = data[i].ClosePrice
			index = i
		}
	}
	return low, index
}

// 辅助函数：在RSI数据中寻找RSI低点
func findRSIHigh(data []*RSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].RSI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].RSI > high {
			high = data[i].RSI
			index = i
		}
	}
	return high, index
}

// 辅助函数：在RSI数据中寻找RSI低点
func findRSILow(data []*RSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].RSI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].RSI < low {
			low = data[i].RSI
			index = i
		}
	}
	return low, index
}

// 在主函数中调用
func GetRSISignal(klines []Kline, period int) Signal {
	// 1.	绝不单独使用RSI：RSI必须与价格行为分析、趋势线、支撑/阻力位以及其他指标（如均线、MACD）结合使用。

	// 2.	适应市场状态：
	// o	震荡市：RSI的超买/超卖信号非常有效。
	// o	强趋势市：RSI可能在超买/超卖区停留很长时间，此时逆势交易会带来巨大亏损。应顺势而为，只在趋势方向信号出现时入场。

	// 3.	背离是最高质量信号：RSI背离（尤其是发生在超买/超卖区的背离）是RSI提供的最高质量的交易信号，胜率远高于简单的超买超卖交叉。

	// 4.	参数调整：标准参数是14期。对于波动更大的BTC，可以尝试更短的参数（如9或6）来让RSI更敏感，或更长的参数（如21或25）来过滤噪音。

	// 5.	多时间框架分析：
	// o	在日线图上识别主要的超买超卖状态和背离。
	// o	在4小时或1小时图上寻找具体的入场时机。

	// 6.	风险管理：
	// o	在RSI超卖区开多，止损应设在价格新低下方。
	// o	在RSI超买区开空，止损应设在价格新高上方。
	// o	基于背离交易时，止损应设在背离起点之外。

	// 计算RSI，通常使用14周期
	rsiData := calculateRSIData(klines, period)

	// 分析RSI信号
	signal := analyzeRSISignal(rsiData, 20)

	return signal
}
