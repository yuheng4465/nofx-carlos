package market

import (
	"fmt"
	"math"
)

// BollingerBandData 存储布林带数据点
type BollingerBandData struct {
	OpenTime   int64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
	MiddleBand float64 // 中轨
	UpperBand  float64 // 上轨
	LowerBand  float64 // 下轨
	BandWidth  float64 // 带宽
	PercentB   float64 // %b指标，表示价格在布林带中的位置
}

// calculateBollingerBands 计算布林带
func calculateBollingerBands(klines []Kline, period int, multiplier float64) []*BollingerBandData {
	var bbData []*BollingerBandData
	var closes []float64

	for i, kline := range klines {
		closePrice := kline.Close
		closes = append(closes, closePrice)

		if i >= period-1 {
			// 计算中轨
			sum := 0.0
			for j := 0; j < period; j++ {
				sum += closes[i-j]
			}
			middleBand := sum / float64(period)

			// 计算标准差
			variance := 0.0
			for j := 0; j < period; j++ {
				diff := closes[i-j] - middleBand
				variance += diff * diff
			}
			stdDev := math.Sqrt(variance / float64(period))

			// 计算上下轨
			upperBand := middleBand + (stdDev * multiplier)
			lowerBand := middleBand - (stdDev * multiplier)

			// 计算带宽和 %b
			bandWidth := (upperBand - lowerBand) / middleBand
			percentB := (closePrice - lowerBand) / (upperBand - lowerBand)

			bbData = append(bbData, &BollingerBandData{
				OpenTime:   kline.OpenTime,
				ClosePrice: closePrice,
				MiddleBand: middleBand,
				UpperBand:  upperBand,
				LowerBand:  lowerBand,
				BandWidth:  bandWidth,
				PercentB:   percentB,
			})
		}
	}
	return bbData
}

// 转换为字符串
func getBOLLDataString(bbData []*BollingerBandData, period int) string {
	var middleBand, upperBand, lowerBand []float64
	// 取尾部数据
	startIndex := len(bbData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := bbData[startIndex:]
	for _, v := range lastData {
		middleBand = append(middleBand, v.MiddleBand)
		upperBand = append(upperBand, v.UpperBand)
		lowerBand = append(lowerBand, v.LowerBand)
	}

	// 将字节切片转换为字符串
	middleBandStr := formatFloatSlice(middleBand)
	upperBandStr := formatFloatSlice(upperBand)
	lowerBandStr := formatFloatSlice(lowerBand)

	return fmt.Sprintf("middleBand: %s , upperBand: %s , lowerBand: %s", middleBandStr, upperBandStr, lowerBandStr)
}

// 计算布林带带宽
func calculateBollingerWidth(bbData []*BollingerBandData) float64 {
	current := bbData[len(bbData)-1]
	return (current.UpperBand - current.LowerBand) / current.MiddleBand * 100
}

// 获取策略所需数据
func getBollingerCases(bbData []*BollingerBandData, lookback int) *Cases {
	if len(bbData) < lookback+1 {
		return &Cases{}
	}
	current := bbData[len(bbData)-1]
	prev := bbData[len(bbData)-2]

	// 带宽
	avgBandWidth := 0.0
	for i := len(bbData) - lookback; i < len(bbData); i++ {
		avgBandWidth += bbData[i].BandWidth
	}
	avgBandWidth /= float64(lookback)

	casesData := &Cases{}
	casesData.Name = TargetMACD
	casesData.Metrics = map[string]interface{}{
		"currentBandWidth":  current.BandWidth,
		"avgBandWidth":      avgBandWidth,
		"prevPrice":         prev.ClosePrice,
		"currentPrice":      current.ClosePrice,
		"prevUpperBand":     prev.UpperBand,
		"currentUpperBand":  current.UpperBand,
		"prevLowerBand":     prev.LowerBand,
		"currentLowerBand":  current.LowerBand,
		"currentMiddleBand": current.MiddleBand,
		"prevMiddleBand":    prev.MiddleBand,
	}

	return casesData
}

// analyzeBollingerSignal 分析布林带数据，生成交易信号
func analyzeBollingerSignal(bbData []*BollingerBandData, lookback int, period string) *Signal {
	if len(bbData) < lookback+1 {
		return &Signal{Target: TargetBOLL, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := bbData[len(bbData)-1]
	prev := bbData[len(bbData)-2]

	// 信号1: 带宽收缩检测 (寻找Squeeze)
	avgBandWidth := 0.0
	for i := len(bbData) - lookback; i < len(bbData); i++ {
		avgBandWidth += bbData[i].BandWidth
	}
	avgBandWidth /= float64(lookback)

	// 如果当前带宽远小于近期平均带宽，则认为处于收缩状态
	isSqueeze := current.BandWidth < avgBandWidth*0.7

	// 信号2: 突破信号
	// 从上轨下方突破到上轨上方
	if prev.ClosePrice <= prev.UpperBand && current.ClosePrice > current.UpperBand {
		if isSqueeze {
			return &Signal{
				Target:     TargetBOLL,
				SignalType: "strong_bullish_breakout",
				Period:     period,
				Side:       SideBuy,
				Confidence: 0.8,
				Message:    "布林带收缩后向上突破上轨！强烈开多信号！",
			}
		}
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bullish_breakout",
			Period:     period,
			Side:       SideBuy,
			Confidence: 0.6,
			Message:    "价格突破上轨，潜在开多信号",
		}
	}

	// 从下轨上方跌破到下轨下方
	if prev.ClosePrice >= prev.LowerBand && current.ClosePrice < current.LowerBand {
		if isSqueeze {
			return &Signal{
				Target:     TargetBOLL,
				SignalType: "strong_bearish_breakout",
				Period:     period,
				Side:       SideSell,
				Confidence: 0.8,
				Message:    "布林带收缩后向下跌破下轨！强烈开空信号！",
			}
		}
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bearish_breakout",
			Period:     period,
			Side:       SideSell,
			Confidence: 0.6,
			Message:    "价格跌破下轨，潜在开空信号",
		}
	}

	// 信号3: 反转信号 (从下轨反弹/从上轨回落)
	// 如果前一根K线接触或跌破下轨，当前K线收高，视为反弹信号
	if prev.ClosePrice <= prev.LowerBand && current.ClosePrice > prev.LowerBand {
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bullish_reversal",
			Period:     period,
			Side:       SideBuy,
			Confidence: 0.7,
			Message:    "价格从下轨反弹，潜在开多信号",
		}
	}

	// 如果前一根K线接触或突破上轨，当前K线收低，视为回落信号
	if prev.ClosePrice >= prev.UpperBand && current.ClosePrice < prev.UpperBand {
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bearish_reversal",
			Period:     period,
			Side:       SideSell,
			Confidence: 0.7,
			Message:    "价格从上轨回落，潜在开空信号",
		}
	}

	// 信号4: 趋势中的中轨支撑/阻力
	if current.ClosePrice > current.MiddleBand {
		// 在上升趋势中，如果价格回调接近中轨
		if current.ClosePrice < current.MiddleBand*1.01 { // 价格在中轨附近1%范围内
			return &Signal{
				Target:     TargetBOLL,
				SignalType: "trend_bullish_pullback",
				Period:     period,
				Side:       SideBuy,
				Confidence: 0.6,
				Message:    "价格在上升趋势中回调至中轨支撑，潜在开多机会",
			}
		}
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bullish_trend",
			Period:     period,
			Side:       SideBuy,
			Confidence: 0.5,
			Message:    "价格在中轨上方，处于上升趋势",
		}
	} else {
		// 在下降趋势中，如果价格反弹接近中轨
		if current.ClosePrice > current.MiddleBand*0.99 { // 价格在中轨附近1%范围内
			return &Signal{
				Target:     TargetBOLL,
				SignalType: "trend_bearish_pullback",
				Period:     period,
				Side:       SideSell,
				Confidence: 0.6,
				Message:    "价格在下降趋势中反弹至中轨阻力，潜在开空机会",
			}
		}
		return &Signal{
			Target:     TargetBOLL,
			SignalType: "bearish_trend",
			Period:     period,
			Side:       SideSell,
			Confidence: 0.5,
			Message:    "价格在中轨下方，处于下降趋势",
		}
	}
}

func getBollingerSignal(klines []Kline, period int, timePeriod string) *Signal {
	// 1、绝不单独使用布林带：布林带必须与其他指标结合使用以过滤假信号：
	// RSI/MACD：用于确认背离。例如，价格突破上轨的同时RSI出现顶背离，则突破失败的概率大增。
	// 成交量：收缩后的突破必须放量，否则可能是假突破。反弹/回落时缩量更佳。
	// K线形态：在支撑/阻力位寻找看涨/看跌的K线组合（如Pin Bar、吞没）进行确认。

	// 2、时间框架选择：
	// - 大周期定方向：日线图判断主要趋势。
	// - 小周期找点位：1小时、4小时图寻找具体的开多/开空入场点。

	// 3、区分趋势与震荡：
	// - 在强劲的单边趋势中，价格会持续“贴着”布林带上轨或下轨运行，此时的“超买/超卖”信号是无效的，应顺势而为。
	// - 在震荡市中，布林带的上下轨反转信号非常有效。

	// 4、风险管理：
	// - 开多止损：设置在信号K线低点下方或下轨下方。
	// - 开空止损：设置在信号K线高点上方的上轨上方。
	// - BandWidth的应用：带宽很窄时，潜在的盈利空间大，可以适当放宽止损；带宽很宽时，波动剧烈，应收紧止损。

	// 计算布林带，标准参数为20期，2倍标准差
	bbData := calculateBollingerBands(klines, period, 2.0)

	// 分析信号
	signal := analyzeBollingerSignal(bbData, 10, timePeriod)

	return signal
}
