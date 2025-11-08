package market

// WMAData 存储WMA数据点
type WMAData struct {
	OpenTime   int64
	ClosePrice float64
	WMA        float64
	WMASlope   float64 // WMA斜率
}

// calculateWMA 计算加权移动平均线
func calculateWMA(klines []Kline, period int) []*WMAData {
	var wmaData []*WMAData

	for i := range klines {
		if i < period-1 {
			// 不足周期时跳过
			wmaData = append(wmaData, &WMAData{
				OpenTime:   klines[i].OpenTime,
				ClosePrice: 0,
				WMA:        0,
				WMASlope:   0,
			})
			continue
		}

		// 计算加权和
		weightedSum := 0.0
		totalWeight := 0.0

		for j := 0; j < period; j++ {
			closePrice := klines[i-j].Close
			weight := float64(period - j) // 近期权重更高
			weightedSum += closePrice * weight
			totalWeight += weight
		}

		wmaValue := weightedSum / totalWeight

		// 计算斜率
		slope := 0.0
		if i > period-1 {
			prevWMA := wmaData[i-1].WMA
			slope = wmaValue - prevWMA
		}

		currentClose := klines[i].Close
		wmaData = append(wmaData, &WMAData{
			OpenTime:   klines[i].OpenTime,
			ClosePrice: currentClose,
			WMA:        wmaValue,
			WMASlope:   slope,
		})
	}
	return wmaData
}

// calculateMultiPeriodWMA 计算多周期WMA
func calculateMultiPeriodWMA(klines []Kline, periods []int) map[int][]*WMAData {
	result := make(map[int][]*WMAData)
	for _, period := range periods {
		result[period] = calculateWMA(klines, period)
	}
	return result
}

// analyzeWMASignal 分析WMA数据，生成交易信号
func analyzeWMASignal(wmaData []*WMAData, multiWMA map[int][]*WMAData) Signal {
	if len(wmaData) < 2 {
		return Signal{Target: "WMA", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := wmaData[len(wmaData)-1]
	prev := wmaData[len(wmaData)-2]

	// 检查多周期WMA数据
	wma10, has10 := multiWMA[10]
	wma30, has30 := multiWMA[30]
	wma50, has50 := multiWMA[50]

	// 信号1: 价格与WMA位置关系
	priceAboveWMA := current.ClosePrice > current.WMA
	priceBelowWMA := current.ClosePrice < current.WMA
	prevPriceAboveWMA := prev.ClosePrice > prev.WMA
	prevPriceBelowWMA := prev.ClosePrice < prev.WMA

	// 价格上穿WMA
	if prevPriceBelowWMA && priceAboveWMA {
		return Signal{
			Target:     "WMA",
			SignalType: "bullish_crossover",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "价格上穿WMA！短期趋势转多",
		}
	}

	// 价格下穿WMA
	if prevPriceAboveWMA && priceBelowWMA {
		return Signal{
			Target:     "WMA",
			SignalType: "bearish_crossover",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "价格下穿WMA！短期趋势转空",
		}
	}

	// 信号2: WMA支撑阻力作用
	if priceAboveWMA {
		// 计算价格与WMA的距离百分比
		distancePercent := (current.ClosePrice - current.WMA) / current.WMA * 100
		if distancePercent < 0.5 { // 价格在WMA附近0.5%范围内
			if current.WMASlope > 0 {
				return Signal{
					Target:     "WMA",
					SignalType: "bullish_support",
					Side:       "buy",
					Confidence: 0.8,
					Message:    "价格在上升的WMA处获得支撑！强烈开多信号",
				}
			}
		}
	}

	if priceBelowWMA {
		distancePercent := (current.WMA - current.ClosePrice) / current.WMA * 100
		if distancePercent < 0.5 { // 价格在WMA附近0.5%范围内
			if current.WMASlope < 0 {
				return Signal{
					Target:     "WMA",
					SignalType: "bearish_resistance",
					Side:       "sell",
					Confidence: 0.8,
					Message:    "价格在下降的WMA处受到阻力！强烈开空信号",
				}
			}
		}
	}

	// 信号3: 多周期WMA排列
	if has10 && has30 && has50 {
		currentWMA10 := wma10[len(wma10)-1].WMA
		currentWMA30 := wma30[len(wma30)-1].WMA
		currentWMA50 := wma50[len(wma50)-1].WMA

		// 多头排列
		if currentWMA10 > currentWMA30 && currentWMA30 > currentWMA50 {
			return Signal{
				Target:     "WMA",
				SignalType: "bullish_alignment",
				Side:       "buy",
				Confidence: 0.9,
				Message:    "WMA多头排列！各周期趋势一致向上，强烈看多",
			}
		}

		// 空头排列
		if currentWMA10 < currentWMA30 && currentWMA30 < currentWMA50 {
			return Signal{
				Target:     "WMA",
				SignalType: "bearish_alignment",
				Side:       "sell",
				Confidence: 0.9,
				Message:    "WMA空头排列！各周期趋势一致向下，强烈看空",
			}
		}

		// 金叉信号 (WMA10上穿WMA30)
		prevWMA10 := wma10[len(wma10)-2].WMA
		prevWMA30 := wma30[len(wma30)-2].WMA
		if prevWMA10 <= prevWMA30 && currentWMA10 > currentWMA30 {
			return Signal{
				Target:     "WMA",
				SignalType: "golden_cross",
				Side:       "buy",
				Confidence: 0.8,
				Message:    "WMA金叉！短期WMA上穿中期WMA，趋势转多",
			}
		}

		// 死叉信号 (WMA10下穿WMA30)
		if prevWMA10 >= prevWMA30 && currentWMA10 < currentWMA30 {
			return Signal{
				Target:     "WMA",
				SignalType: "dead_cross",
				Side:       "sell",
				Confidence: 0.8,
				Message:    "WMA死叉！短期WMA下穿中期WMA，趋势转空",
			}
		}
	}

	// 信号4: 趋势强度判断
	if priceAboveWMA && current.WMASlope > 0 {
		return Signal{
			Target:     "WMA",
			SignalType: "strong_bullish_trend",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "价格在WMA上方且WMA斜率向上，强势上涨",
		}
	}

	if priceBelowWMA && current.WMASlope < 0 {
		return Signal{
			Target:     "WMA",
			SignalType: "strong_bearish_trend",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "价格在WMA下方且WMA斜率向下，强势下跌",
		}
	}

	return Signal{Target: "WMA", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确信号"}
}

func GetWMASignal(klines []Kline, period int) Signal {
	// 1.	WMA的参数选择：
	// o	短期WMA：5-15周期，用于捕捉短期趋势和入场点
	// o	中期WMA：20-30周期，用于确认趋势方向
	// o	长期WMA：50-100周期，用于识别主要趋势

	// 2.	结合其他指标过滤信号：
	// o	RSI：避免在超买区追多、超卖区追空
	// o	MACD：确认动量变化
	// o	布林带：识别波动性和价格位置
	// o	成交量：确认突破的真实性

	// 3.	WMA的敏感性既是优点也是缺点：
	// o	优点：能快速识别趋势变化，适合短线交易
	// o	缺点：在震荡市中容易产生假信号，需要其他指标确认

	// 4.	多时间框架分析：
	// o	日线图：用WMA50/WMA100判断主要趋势
	// o	4小时图：用WMA20/WMA30寻找交易方向
	// o	1小时图：用WMA10精确定位入场点

	// 5.	风险管理：
	// o	做多止损：设置在WMA下方或近期摆动低点
	// o	做空止损：设置在WMA上方或近期摆动高点
	// o	当价格远离WMA时，意味着短期波动过大，不宜追单

	// 6.	WMA与其他移动平均线的结合：
	// o	可以同时使用WMA和EMA，WMA提供敏感信号，EMA提供确认
	// o	WMA金叉/死叉比SMA出现更早，但需要成交量确认

	// 计算多周期WMA
	periods := []int{10, 30, 50} // 短期、中期、长期WMA
	multiWMA := calculateMultiPeriodWMA(klines, periods)

	// 使用WMA10作为主要分析对象
	wma10Data := multiWMA[10]

	// 分析WMA信号
	signal := analyzeWMASignal(wma10Data, multiWMA)

	return signal

}
