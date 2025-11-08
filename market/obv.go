package market

import (
	"math"
)

// CalculateOBV 计算能量潮
func CalculateOBV(klines []Kline) []OBVData {
	results := make([]OBVData, len(klines))

	// 第一天的OBV为0
	results[0] = OBVData{
		OpenTime: klines[0].OpenTime,
		OBV:      0,
	}

	// 计算后续日期的OBV
	for i := 1; i < len(klines); i++ {
		prevOBV := results[i-1].OBV
		currentClose := klines[i].Close
		prevClose := klines[i-1].Close
		currentVolume := klines[i].Volume

		var currentOBV float64

		switch {
		case currentClose > prevClose:
			currentOBV = prevOBV + currentVolume
		case currentClose < prevClose:
			currentOBV = prevOBV - currentVolume
		default:
			currentOBV = prevOBV
		}

		results[i] = OBVData{
			OpenTime:   klines[i].OpenTime,
			OBV:        currentOBV,
			ClosePrice: klines[i].Close,
		}
	}

	return results
}

// analyzeOBVWithPeaks 分析OBV，寻找价格和OBV的峰值/谷值以探测背离
func analyzeOBVWithPeaks(obvData []OBVData, lookbackPeriod int) Signal {
	if len(obvData) < lookbackPeriod*2 {
		return Signal{Target: "OBV", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	// 寻找价格的最近高点和低点
	recentPriceHigh, recentPriceHighIndex := findOBVPriceHigh(obvData, lookbackPeriod)
	recentPriceLow, recentPriceLowIndex := findOBVPriceLow(obvData, lookbackPeriod)

	// 寻找OBV的对应高点和低点
	recentOBVHigh, recentOBVHighIndex := findOBVHigh(obvData, lookbackPeriod)
	recentOBVLow, recentOBVLowIndex := findOBVLow(obvData, lookbackPeriod)

	// 1. 检查顶背离：价格创新高，但OBV没有
	if recentPriceHighIndex > len(obvData)-5 { // 如果价格高点非常近期
		if recentOBVHighIndex < recentPriceHighIndex { // 且OBV高点较早出现
			pricePrevHigh, _ := findOBVPriceHigh(obvData[:recentPriceHighIndex], lookbackPeriod)
			obvPrevHigh, _ := findOBVPriceHigh(obvData[:recentOBVHighIndex], lookbackPeriod)

			// 如果价格创新高，但OBV高点下降
			if recentPriceHigh > pricePrevHigh && recentOBVHigh < obvPrevHigh {
				return Signal{
					Target:     "OBV",
					SignalType: "bearish_divergence",
					Side:       "sell",
					Confidence: 0.8, // 可以基于背离程度计算置信度
					Message:    "检测到顶背离：价格创新高但OBV下降，是潜在的开空信号",
				}
			}
		}
	}

	// 2. 检查底背离：价格创新低，但OBV没有
	if recentPriceLowIndex > len(obvData)-5 { // 如果价格低点非常近期
		if recentOBVLowIndex < recentPriceLowIndex { // 且OBV低点较早出现
			pricePrevLow, _ := findOBVPriceLow(obvData[:recentPriceLowIndex], lookbackPeriod)
			obvPrevLow, _ := findOBVPriceLow(obvData[:recentOBVLowIndex], lookbackPeriod)

			// 如果价格创新低，但OBV低点抬高
			if recentPriceLow < pricePrevLow && recentOBVLow > obvPrevLow {
				return Signal{
					Target:     "OBV",
					SignalType: "bullish_divergence",
					Side:       "buy",
					Confidence: 0.8,
					Message:    "检测到底背离：价格创新低但OBV抬高，是潜在的开多信号",
				}
			}
		}
	}

	// 3. 检查OBV整体趋势
	obvTrend := checkOBVTrend(obvData, 20)
	if obvTrend > 0.5 {
		return Signal{
			Target:     "OBV",
			SignalType: "trend_bullish",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "OBV处于强劲上升趋势，建议寻找开多机会",
		}
	} else if obvTrend < -0.5 {
		return Signal{
			Target:     "OBV",
			SignalType: "trend_bearish",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "OBV处于下降趋势，建议寻找开空机会",
		}
	}

	return Signal{Target: "OBV", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确信号"}
}

// 辅助函数：寻找OBV高点
func findOBVHigh(data []OBVData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].OBV
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].OBV > high {
			high = data[i].OBV
			index = i
		}
	}
	return high, index
}

// 辅助函数：寻找OBV低点
func findOBVLow(data []OBVData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].OBV
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].OBV < low {
			low = data[i].OBV
			index = i
		}
	}
	return low, index
}

// 辅助函数：寻找价格高点
func findOBVPriceHigh(data []OBVData, lookback int) (float64, int) {
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

// 辅助函数：寻找价格低点
func findOBVPriceLow(data []OBVData, lookback int) (float64, int) {
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

// 检查OBV整体趋势
func checkOBVTrend(data []OBVData, period int) float64 {
	if len(data) < period {
		return 0
	}
	startOBV := data[len(data)-period].OBV
	endOBV := data[len(data)-1].OBV
	// 返回趋势强度，正数为上升，负数为下降
	return (endOBV - startOBV) / math.Abs(startOBV) * 100
}

func GetObvSignal(obvData []OBVData) Signal {
	// 分析信号
	signal := analyzeOBVWithPeaks(obvData, 10)

	return signal
}
