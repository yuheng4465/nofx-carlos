package market

// EMAData 存储EMA数据点
type EMAData struct {
	OpenTime   int64
	ClosePrice float64
	EMA20      float64
	EMASlope   float64 // EMA20的斜率
}

// calculateEMA 计算指数移动平均线
func calculateEMAData(klines []Kline, period int) []*EMAData {
	var emaData []*EMAData
	var closes []float64
	multiplier := 2.0 / (float64(period) + 1.0)

	for i, kline := range klines {
		closePrice := kline.Close
		closes = append(closes, closePrice)

		var emaValue float64
		if i == 0 {
			// 第一天的EMA就是当天的收盘价
			emaValue = closePrice
		} else if i < period {
			// 在计算周期内，使用简单移动平均
			sum := 0.0
			for j := 0; j <= i; j++ {
				sum += closes[j]
			}
			emaValue = sum / float64(i+1)
		} else {
			// 使用EMA公式计算
			prevEMA := emaData[i-1].EMA20
			emaValue = (closePrice-prevEMA)*multiplier + prevEMA
		}

		// 计算斜率 (当前EMA - 前一期EMA)
		slope := 0.0
		if i > 0 {
			slope = emaValue - emaData[i-1].EMA20
		}

		emaData = append(emaData, &EMAData{
			OpenTime:   kline.OpenTime,
			ClosePrice: closePrice,
			EMA20:      emaValue,
			EMASlope:   slope,
		})
	}
	return emaData
}

// analyzeEMASignal 分析EMA20数据，生成交易信号
func analyzeEMASignal(emaData []*EMAData, lookback int) Signal {
	if len(emaData) < lookback+1 {
		return Signal{Target: "EMA20", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := emaData[len(emaData)-1]
	prev := emaData[len(emaData)-2]

	// 判断价格与EMA20的位置关系
	priceAboveEMA := current.ClosePrice > current.EMA20
	priceBelowEMA := current.ClosePrice < current.EMA20
	prevPriceAboveEMA := prev.ClosePrice > prev.EMA20
	prevPriceBelowEMA := prev.ClosePrice < prev.EMA20

	// 信号1: 价格穿越EMA20
	// 从下方上穿EMA20
	if prevPriceBelowEMA && priceAboveEMA {
		return Signal{
			Target:     "EMA20",
			SignalType: "bullish_crossover",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "价格上穿EMA20！短期趋势转多，潜在开多信号",
		}
	}

	// 从上下穿EMA20
	if prevPriceAboveEMA && priceBelowEMA {
		return Signal{
			Target:     "EMA20",
			SignalType: "bearish_crossover",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "价格下穿EMA20！短期趋势转空，潜在开空信号",
		}
	}

	// 信号2: EMA20支撑阻力作用
	// 价格在EMA20上方且回调受到支撑
	if priceAboveEMA {
		// 计算价格距离EMA20的百分比
		distancePercent := (current.ClosePrice - current.EMA20) / current.EMA20 * 100
		if distancePercent < 1.0 { // 价格在EMA20附近1%范围内
			if current.EMASlope > 0 {
				return Signal{
					Target:     "EMA20",
					SignalType: "bullish_support",
					Side:       "buy",
					Confidence: 0.8,
					Message:    "价格在上升的EMA20处获得支撑！强烈开多信号",
				}
			}
			return Signal{
				Target:     "EMA20",
				SignalType: "bullish_near_support",
				Side:       "buy",
				Confidence: 0.6,
				Message:    "价格在EMA20支撑附近，关注做多机会",
			}
		}
	}

	// 价格在EMA20下方且反弹受到阻力
	if priceBelowEMA {
		distancePercent := (current.EMA20 - current.ClosePrice) / current.EMA20 * 100
		if distancePercent < 1.0 { // 价格在EMA20附近1%范围内
			if current.EMASlope < 0 {
				return Signal{
					Target:     "EMA20",
					SignalType: "bearish_resistance",
					Side:       "sell",
					Confidence: 0.8,
					Message:    "价格在下降的EMA20处受到阻力！强烈开空信号",
				}
			}
			return Signal{
				Target:     "EMA20",
				SignalType: "bearish_near_resistance",
				Side:       "sell",
				Confidence: 0.6,
				Message:    "价格在EMA20阻力附近，关注做空机会",
			}
		}
	}

	// 信号3: 趋势强度判断
	if priceAboveEMA && current.EMASlope > 0 {
		return Signal{
			Target:     "EMA20",
			SignalType: "strong_bullish_trend",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "价格在EMA20上方且EMA斜率向上，强势上涨趋势",
		}
	}

	if priceBelowEMA && current.EMASlope < 0 {
		return Signal{
			Target:     "EMA20",
			SignalType: "strong_bearish_trend",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "价格在EMA20下方且EMA斜率向下，强势下跌趋势",
		}
	}

	return Signal{Target: "EMA20", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确信号，建议观望"}
}

// 获取信号
func GetEMASignal(klines []Kline, period int) Signal {
	// 1.	结合其他指标过滤假信号：EMA20在趋势市中表现优异，但在震荡市中会产生大量“拉锯”信号。务必结合以下工具：
	// o	布林带：在布林带收缩后突破时，EMA20的穿越信号更可靠。
	// o	MACD/RSI：用于确认动量和超买超卖状态。
	// o	成交量：突破EMA20时放量，回测EMA20时缩量，是高质量信号的标志。

	// 2.	多时间框架分析：
	// o	日线图：用EMA20判断主要趋势方向。
	// o	4小时/1小时图：寻找具体的入场点位。
	// o	原则：在大周期趋势的方向上，交易小周期的信号。

	// 3.	EMA20的滞后性：作为移动平均线，EMA20本质上是滞后指标。它确认趋势很好，但捕捉趋势起点较晚。不要指望买在最低点、卖在最高点。

	// 4.	风险管理：
	// o	做多止损：设置在EMA20下方或近期摆动低点。
	// o	做空止损：设置在EMA20上方或近期摆动高点。
	// o	当价格远离EMA20时，意味着短期涨幅过大，追涨杀跌的风险增加。

	// 5.	结合EMA排列：单独使用EMA20有时会显得单薄。如果能结合EMA50和EMA200形成多头/空头排列，信号的可靠性会成倍增加。

	// 计算EMA20
	emaData := calculateEMAData(klines, period)

	// 分析信号
	signal := analyzeEMASignal(emaData, 5)

	return signal
}
