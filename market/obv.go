package market

import (
	"math"
)

// OBVData 存储成交量数据点
type OBVData struct {
	OpenTime   int64
	ClosePrice float64
	Volume     float64
	OBV        float64
}

// CalculateOBV 计算能量潮
func calculateOBV(klines []Kline) []*OBVData {
	// 检查输入数据是否有效
	if len(klines) == 0 {
		return nil
	}
	// 正确初始化一个空的切片，而不是一个充满nil的切片
	var results []*OBVData

	// 第一天的OBV为0
	results = append(results, &OBVData{
		OpenTime: klines[0].OpenTime,
		OBV:      0,
	})

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

		results = append(results, &OBVData{
			OpenTime:   klines[i].OpenTime,
			OBV:        currentOBV,
			ClosePrice: klines[i].Close,
		})
	}

	return results
}

// 转换为字符串
func getOBVDataString(obvData []*OBVData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(obvData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := obvData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.OBV)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// analyzeOBVWithPeaks 分析OBV，寻找价格和OBV的峰值/谷值以探测背离
func analyzeOBVWithPeaks(obvData []*OBVData, lookbackPeriod int, period string) *Signal {
	if len(obvData) < lookbackPeriod*2 {
		return &Signal{Target: TargetOBV, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
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
				return &Signal{
					Target:     TargetOBV,
					SignalType: "bearish_divergence",
					Side:       SideSell,
					Period:     period,
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
				return &Signal{
					Target:     TargetOBV,
					SignalType: "bullish_divergence",
					Side:       SideBuy,
					Period:     period,
					Confidence: 0.8,
					Message:    "检测到底背离：价格创新低但OBV抬高，是潜在的开多信号",
				}
			}
		}
	}

	// 3. 检查OBV整体趋势
	obvTrend := checkOBVTrend(obvData, 20)
	if obvTrend > 0.5 {
		return &Signal{
			Target:     TargetOBV,
			SignalType: "trend_bullish",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "OBV处于强劲上升趋势，建议寻找开多机会",
		}
	} else if obvTrend < -0.5 {
		return &Signal{
			Target:     TargetOBV,
			SignalType: "trend_bearish",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "OBV处于下降趋势，建议寻找开空机会",
		}
	}

	return &Signal{Target: TargetOBV, SignalType: "none", Side: SideNone, Period: period, Confidence: 0.5, Message: "未发现明确信号"}
}

// 辅助函数：寻找OBV高点
func findOBVHigh(data []*OBVData, lookback int) (float64, int) {
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
func findOBVLow(data []*OBVData, lookback int) (float64, int) {
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
func findOBVPriceHigh(data []*OBVData, lookback int) (float64, int) {
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
func findOBVPriceLow(data []*OBVData, lookback int) (float64, int) {
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
func checkOBVTrend(data []*OBVData, period int) float64 {
	if len(data) < period {
		return 0
	}
	startOBV := data[len(data)-period].OBV
	endOBV := data[len(data)-1].OBV
	// 返回趋势强度，正数为上升，负数为下降
	return (endOBV - startOBV) / math.Abs(startOBV) * 100
}

func getObvSignal(obvData []*OBVData, period string) *Signal {
	// 1.	绝不单独使用OBV：OBV信号必须与价格行为、K线形态（如吞没、pin bar）和其他指标（如均线、布林带、MACD）结合使用，相互验证后再行动。
	// 2.	时间框架选择：
	// o	日线/周线上的背离信号比1小时/15分钟上的信号更可靠。
	// o	建议采用多时间框架分析：在大周期（如日线）确定主要方向，在小周期（如1小时）寻找具体入场点。

	// 3.	风险管理：无论OBV信号看起来多么完美，都必须设置止损单。例如，在开多时，将止损设置在近期低点下方；开空时，将止损设置在近期高点上方。
	// 4.	结合趋势：在上升趋势中，OBV的"开多"信号更可信；在下降趋势中，OBV的"开空"信号更可信。不要逆势交易。

	// 分析信号
	signal := analyzeOBVWithPeaks(obvData, 10, period)

	return signal
}
