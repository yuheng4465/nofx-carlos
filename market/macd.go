package market

// MACDData 存储MACD数据点
type MACDData struct {
	OpenTime       int64
	ClosePrice     float64
	MACDLine       float64 // DIF线
	SignalLine     float64 // DEA线
	Histogram      float64 // 柱状线
	HistogramColor string  // 柱状线颜色
}

// calculateMACDData 计算MACD指标
func calculateMACDData(klines []Kline, fastPeriod, slowPeriod, signalPeriod int) []*MACDData {
	if len(klines) == 0 {
		return nil
	}

	var macdData []*MACDData
	var emaFast, emaSlow float64

	// 收盘价、最高价、最低价
	// var closes []float64
	// for _, kline := range klines {
	// 	closes = append(closes, kline.Close)
	// }

	// outMACD, outMACDSignal, outMACDHist := talib.Macd(closes, fastPeriod, slowPeriod, signalPeriod)
	// for i, macd := range outMACD {
	// 	data := &MACDData{
	// 		ClosePrice: closes[i],
	// 		MACDLine:   macd,
	// 		SignalLine: outMACDSignal[i],
	// 		Histogram:  outMACDHist[i],
	// 	}

	// 	macdData = append(macdData, data)
	// }

	// return macdData

	for i, kline := range klines {
		closePrice := kline.Close

		// 计算EMA12和EMA26
		if i == 0 {
			emaFast = closePrice
			emaSlow = closePrice
		} else {
			// 计算快速EMA
			multiplierFast := 2.0 / (float64(fastPeriod) + 1.0)
			emaFast = (closePrice-emaFast)*multiplierFast + emaFast

			// 计算慢速EMA
			multiplierSlow := 2.0 / (float64(slowPeriod) + 1.0)
			emaSlow = (closePrice-emaSlow)*multiplierSlow + emaSlow
		}

		// 计算MACD线
		macdLine := emaFast - emaSlow

		// 计算信号线
		var signalLine float64
		if i == 0 {
			signalLine = macdLine
		} else if i < signalPeriod {
			// 确保macdData有足够的元素
			if len(macdData) <= i {
				signalLine = macdLine // 备用方案
				continue
			}

			sum := 0.0
			for j := 0; j <= i; j++ {
				if j < len(macdData) {
					sum += macdData[j].MACDLine
				}
			}
			signalLine = sum / float64(i+1)
		} else {
			// 确保有前一个信号线
			if i-1 < len(macdData) && macdData[i-1] != nil {
				prevSignalLine := macdData[i-1].SignalLine
				multiplierSignal := 2.0 / (float64(signalPeriod) + 1.0)
				signalLine = (macdLine-prevSignalLine)*multiplierSignal + prevSignalLine
			} else {
				signalLine = macdLine // 备用方案
			}
		}

		// 计算柱状图
		histogram := macdLine - signalLine
		histogramColor := "green"
		if histogram < 0 {
			histogramColor = "red"
		}

		data := &MACDData{
			OpenTime:       kline.OpenTime,
			ClosePrice:     closePrice,
			MACDLine:       macdLine,
			SignalLine:     signalLine,
			Histogram:      histogram,
			HistogramColor: histogramColor,
		}

		macdData = append(macdData, data)
	}
	return macdData
}

// 转换为字符串
func getMACDDataString(macdData []*MACDData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(macdData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := macdData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.MACDLine)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// 获取策略所需数据
func getMACDCases(macdData []*MACDData, lookback int) *Cases {
	if len(macdData) < lookback+1 {
		return &Cases{}
	}

	current := macdData[len(macdData)-1]
	prev := macdData[len(macdData)-2]

	currentMACD := current.MACDLine
	prevMACD := prev.MACDLine

	currentMACDSignal := current.SignalLine
	prevMACDSignal := prev.SignalLine

	currentMACDHist := current.Histogram
	prevMACDHist := prev.Histogram

	// 背离数据
	recentPriceLow, prevPriceLow, recentMACDLow, prevMACDLow := detectMACDBullishDivergenceData(macdData, 20)
	recentPriceHigh, prevPriceHigh, recentMACDHigh, prevMACDHigh := detectMACDBearishDivergenceData(macdData, 20)

	casesData := &Cases{}
	casesData.Name = TargetMACD
	casesData.Metrics = map[string]interface{}{
		"currentMACD":       currentMACD,
		"prevMACD":          prevMACD,
		"currentMACDSignal": currentMACDSignal,
		"prevMACDSignal":    prevMACDSignal,
		"currentMACDHist":   currentMACDHist,
		"prevMACDHist":      prevMACDHist,
		"recentPriceLow":    recentPriceLow,
		"prevPriceLow":      prevPriceLow,
		"recentMACDLow":     recentMACDLow,
		"prevMACDLow":       prevMACDLow,
		"recentPriceHigh":   recentPriceHigh,
		"prevPriceHigh":     prevPriceHigh,
		"recentMACDHigh":    recentMACDHigh,
		"prevMACDHigh":      prevMACDHigh,
	}

	return casesData
}

// detectMACDBullishDivergence 检测MACD底背离
func detectMACDBullishDivergenceData(macdData []*MACDData, lookback int) (float64, float64, float64, float64) {
	if len(macdData) < lookback*2 {
		return 0, 0, 0, 0
	}

	// 寻找价格低点和MACD低点
	recentPriceLow, recentPriceLowIndex := findPriceLowMACD(macdData, lookback)
	recentMACDLow, recentMACDLowIndex := findMACDHistogramLow(macdData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentMACDLowIndex < 5 {
		return 0, 0, 0, 0
	}

	prevPriceLow, _ := findPriceLowMACD(macdData[:recentPriceLowIndex], 5)
	prevMACDLow, _ := findMACDHistogramLow(macdData[:recentMACDLowIndex], 5)

	return recentPriceLow, prevPriceLow, recentMACDLow, prevMACDLow
}

// detectMACDBearishDivergence 检测MACD顶背离
func detectMACDBearishDivergenceData(macdData []*MACDData, lookback int) (float64, float64, float64, float64) {
	if len(macdData) < lookback*2 {
		return 0, 0, 0, 0
	}

	// 寻找价格高点和MACD高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighMACD(macdData, lookback)
	recentMACDHigh, recentMACDHighIndex := findMACDHistogramHigh(macdData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentMACDHighIndex < 5 {
		return 0, 0, 0, 0
	}

	prevPriceHigh, _ := findPriceHighMACD(macdData[:recentPriceHighIndex], 5)
	prevMACDHigh, _ := findMACDHistogramHigh(macdData[:recentMACDHighIndex], 5)

	// 检查背离：价格创新高，但MACD柱状线高点降低

	return recentPriceHigh, prevPriceHigh, recentMACDHigh, prevMACDHigh
}

// analyzeMACDSignal 分析MACD数据，生成交易信号
func analyzeMACDSignal(macdData []*MACDData, lookback int, period string) *Signal {
	if len(macdData) < lookback+1 {
		return &Signal{Target: TargetMACD, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := macdData[len(macdData)-1]
	prev := macdData[len(macdData)-2]

	// 信号1: 金叉死叉判断
	// 金叉: MACD线从下向上穿过信号线
	if prev.MACDLine <= prev.SignalLine && current.MACDLine > current.SignalLine {
		if current.MACDLine > 0 {
			return &Signal{
				Target:     TargetMACD,
				SignalType: "bullish_golden_cross_above_zero",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.8,
				Message:    "零轴上金叉！强烈开多信号！",
			}
		} else {
			return &Signal{
				Target:     TargetMACD,
				SignalType: "bullish_golden_cross_below_zero",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.6,
				Message:    "零轴下金叉，潜在反弹机会，需谨慎",
			}
		}
	}

	// 死叉: MACD线从上向下穿过信号线
	if prev.MACDLine >= prev.SignalLine && current.MACDLine < current.SignalLine {
		if current.MACDLine < 0 {
			return &Signal{
				Target:     TargetMACD,
				SignalType: "bearish_dead_cross_below_zero",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.8,
				Message:    "零轴下死叉！强烈开空信号！",
			}
		} else {
			return &Signal{
				Target:     TargetMACD,
				SignalType: "bearish_dead_cross_above_zero",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.6,
				Message:    "零轴上死叉，可能回调，需谨慎",
			}
		}
	}

	// 信号2: 零轴穿越
	if prev.MACDLine <= 0 && current.MACDLine > 0 {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "bullish_zero_cross",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.7,
			Message:    "MACD上穿零轴！市场转多，开多信号",
		}
	}

	if prev.MACDLine >= 0 && current.MACDLine < 0 {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "bearish_zero_cross",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.7,
			Message:    "MACD下穿零轴！市场转空，开空信号",
		}
	}

	// 信号3: 柱状线动量分析
	if current.Histogram > 0 && current.Histogram > prev.Histogram {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "bullish_momentum",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "MACD柱状线向上加速，上涨动量增强",
		}
	}

	if current.Histogram < 0 && current.Histogram < prev.Histogram {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "bearish_momentum",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "MACD柱状线向下加速，下跌动量增强",
		}
	}

	// 信号4: 背离检测 (需要更多数据)
	bullishDivergence := detectMACDBullishDivergence(macdData, lookback)
	bearishDivergence := detectMACDBearishDivergence(macdData, lookback)

	if bullishDivergence {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "strong_bullish_divergence",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.9,
			Message:    "发现MACD底背离！强烈开多信号！",
		}
	}

	if bearishDivergence {
		return &Signal{
			Target:     TargetMACD,
			SignalType: "strong_bearish_divergence",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.9,
			Message:    "发现MACD顶背离！强烈开空信号！",
		}
	}

	return &Signal{Target: TargetMACD, SignalType: "none", Side: SideNone, Period: period, Confidence: 0.5, Message: "未发现明确MACD信号"}
}

// detectMACDBullishDivergence 检测MACD底背离
func detectMACDBullishDivergence(macdData []*MACDData, lookback int) bool {
	if len(macdData) < lookback*2 {
		return false
	}

	// 寻找价格低点和MACD低点
	recentPriceLow, recentPriceLowIndex := findPriceLowMACD(macdData, lookback)
	recentMACDLow, recentMACDLowIndex := findMACDHistogramLow(macdData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentMACDLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowMACD(macdData[:recentPriceLowIndex], 5)
	prevMACDLow, _ := findMACDHistogramLow(macdData[:recentMACDLowIndex], 5)

	// 检查背离：价格创新低，但MACD柱状线低点抬高
	if recentPriceLow < prevPriceLow && recentMACDLow > prevMACDLow {
		return true
	}

	return false
}

// detectMACDBearishDivergence 检测MACD顶背离
func detectMACDBearishDivergence(macdData []*MACDData, lookback int) bool {
	if len(macdData) < lookback*2 {
		return false
	}

	// 寻找价格高点和MACD高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighMACD(macdData, lookback)
	recentMACDHigh, recentMACDHighIndex := findMACDHistogramHigh(macdData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentMACDHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighMACD(macdData[:recentPriceHighIndex], 5)
	prevMACDHigh, _ := findMACDHistogramHigh(macdData[:recentMACDHighIndex], 5)

	// 检查背离：价格创新高，但MACD柱状线高点降低
	if recentPriceHigh > prevPriceHigh && recentMACDHigh < prevMACDHigh {
		return true
	}

	return false
}

// 辅助函数：寻找MACD价格高点
func findPriceHighMACD(data []*MACDData, lookback int) (float64, int) {
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

// 辅助函数：在MACD数据中寻找价格低点
func findPriceLowMACD(data []*MACDData, lookback int) (float64, int) {
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

// 辅助函数：在MACD数据中寻找MACD柱状线高点
func findMACDHistogramHigh(data []*MACDData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].Histogram
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].Histogram > high {
			high = data[i].Histogram
			index = i
		}
	}
	return high, index
}

// 辅助函数：在MACD数据中寻找MACD柱状线低点
func findMACDHistogramLow(data []*MACDData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].Histogram
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].Histogram < low {
			low = data[i].Histogram
			index = i
		}
	}
	return low, index
}

// 获取MACD信号
func getMACDSignal(klines []Kline, fastPeriod, slowPeriod, signalPeriod int, timePeriod string) *Signal {
	// 1.	结合其他指标使用：MACD在趋势市中表现优异，但在震荡市中会产生大量假信号。务必结合：
	// o	RSI：确认超买超卖状态
	// o	布林带：识别波动性和价格位置
	// o	成交量：确认突破和趋势强度

	// 2.	多时间框架分析：
	// o	日线图：判断主要趋势方向
	// o	4小时图：寻找主要交易信号
	// o	1小时图：精确定位入场点

	// 3.	理解信号的强弱等级：
	// o	最强信号：背离 + 金叉/死叉
	// o	强信号：零轴上的金叉 / 零轴下的死叉
	// o	中等信号：零轴穿越
	// o	弱信号：零轴下的金叉 / 零轴上的死叉

	// 4.	参数优化：标准参数是(12, 26, 9)。对于波动性更大的BTC，可以尝试：
	// o	更敏感的参数：(6, 13, 5)
	// o	更稳定的参数：(24, 52, 18)

	// 5.	柱状图的先行作用：柱状图的变化通常领先于MACD线的金叉死叉。当柱状线开始拐头时，就要警惕可能出现的金叉/死叉。

	// 6.	风险管理：
	// o	基于MACD信号交易时，止损应设置在最近摆动低点/高点
	// o	背离信号的止损应设置在背离起点之外
	// o	始终控制仓位，使用合理的风险回报比

	// 计算MACD，标准参数(12, 26, 9)
	macdData := calculateMACDData(klines, fastPeriod, slowPeriod, signalPeriod)

	// 分析MACD信号
	signal := analyzeMACDSignal(macdData, 20, timePeriod)

	return signal
}
