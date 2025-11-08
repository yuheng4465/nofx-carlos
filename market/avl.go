package market

// AVLData 存储AVL数据点
type AVLData struct {
	OpenTime         int64
	ClosePrice       float64
	HighPrice        float64
	LowPrice         float64
	Volume           float64
	TypicalPrice     float64 // 典型价格
	CumulativeTPV    float64 // 累计典型价格*成交量
	CumulativeVolume float64 // 累计成交量
	AVL              float64 // 均价线
}

// calculateAVL 计算均价线
// 注意：这里使用VWAP计算方法作为AVL的近似
func calculateAVL(klines []Kline) []*AVLData {
	var avlData []*AVLData
	var cumulativeTPV float64
	var cumulativeVol float64

	for _, kline := range klines {
		// 解析K线数据
		high := kline.High
		low := kline.Low
		close := kline.Close
		volume := kline.Volume

		// 计算典型价格
		typicalPrice := (high + low + close) / 3.0
		// 计算典型价格 * 成交量
		tpv := typicalPrice * volume

		// 更新累计值
		cumulativeTPV += tpv
		cumulativeVol += volume

		// 计算AVL
		avlValue := cumulativeTPV / cumulativeVol

		avlData = append(avlData, &AVLData{
			OpenTime:         kline.OpenTime,
			ClosePrice:       close,
			HighPrice:        high,
			LowPrice:         low,
			Volume:           volume,
			TypicalPrice:     typicalPrice,
			CumulativeTPV:    cumulativeTPV,
			CumulativeVolume: cumulativeVol,
			AVL:              avlValue,
		})
	}
	return avlData
}

// analyzeAVLSignal 分析AVL数据，生成交易信号
func analyzeAVLSignal(avlData []*AVLData, lookback int) Signal {
	if len(avlData) < lookback+1 {
		return Signal{Target: "AVL", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := avlData[len(avlData)-1]
	prev := avlData[len(avlData)-2]

	// 计算AVL斜率
	avlList := make([]float64, len(avlData))
	for i, vwap := range avlData {
		avlList[i] = vwap.AVL
	}
	avlSlope, _ := LinearRegressionAnalysis(avlList)

	// 信号1: 价格与AVL位置关系
	priceAboveAVL := current.ClosePrice > current.AVL
	priceBelowAVL := current.ClosePrice < current.AVL
	prevPriceAboveAVL := prev.ClosePrice > prev.AVL
	prevPriceBelowAVL := prev.ClosePrice < prev.AVL

	// 价格上穿AVL
	if prevPriceBelowAVL && priceAboveAVL {
		return Signal{
			Target:     "AVL",
			SignalType: "bullish_breakthrough",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "价格突破均价线！市场由亏转盈，强烈开多信号",
		}
	}

	// 价格下穿AVL
	if prevPriceAboveAVL && priceBelowAVL {
		return Signal{
			Target:     "AVL",
			SignalType: "bearish_breakdown",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "价格跌破均价线！市场由盈转亏，强烈开空信号",
		}
	}

	// 信号2: AVL支撑阻力作用
	if priceAboveAVL {
		// 计算价格与AVL的距离百分比
		distancePercent := (current.ClosePrice - current.AVL) / current.AVL * 100
		if distancePercent < 1.0 { // 价格在AVL附近1%范围内
			if avlSlope > 0 {
				return Signal{
					Target:     "AVL",
					SignalType: "bullish_cost_support",
					Side:       "buy",
					Confidence: 0.9,
					Message:    "价格在上升的均价线处获得成本支撑！极佳开多机会",
				}
			}
			return Signal{
				Target:     "AVL",
				SignalType: "bullish_near_support",
				Side:       "buy",
				Confidence: 0.7,
				Message:    "价格在均价线支撑附近，关注做多机会",
			}
		}
	}

	if priceBelowAVL {
		distancePercent := (current.AVL - current.ClosePrice) / current.AVL * 100
		if distancePercent < 1.0 { // 价格在AVL附近1%范围内
			if avlSlope < 0 {
				return Signal{
					Target:     "AVL",
					SignalType: "bearish_cost_resistance",
					Side:       "sell",
					Confidence: 0.9,
					Message:    "价格在下降的均价线处受到成本阻力！极佳开空机会",
				}
			}
			return Signal{
				Target:     "AVL",
				SignalType: "bearish_near_resistance",
				Side:       "sell",
				Confidence: 0.7,
				Message:    "价格在均价线阻力附近，关注做空机会",
			}
		}
	}

	// 信号3: 均值回归信号
	// 价格大幅偏离AVL后的回归机会
	distanceFromAVL := (current.ClosePrice - current.AVL) / current.AVL * 100

	if distanceFromAVL > 5.0 { // 价格高于AVL 5%以上
		return Signal{
			Target:     "AVL",
			SignalType: "overextended_bullish",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "价格大幅高于均价线，警惕均值回归回调",
		}
	}

	if distanceFromAVL < -5.0 { // 价格低于AVL 5%以上
		return Signal{
			Target:     "AVL",
			SignalType: "overextended_bearish",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "价格大幅低于均价线，关注均值回归反弹",
		}
	}

	// 信号4: 趋势判断
	if priceAboveAVL && avlSlope > 0 {
		return Signal{
			Target:     "AVL",
			SignalType: "strong_bullish_trend",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "价格在均价线上方且均价线上升，牛市格局健康",
		}
	}

	if priceBelowAVL && avlSlope < 0 {
		return Signal{
			Target:     "AVL",
			SignalType: "strong_bearish_trend",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "价格在均价线下方且均价线下降，熊市格局确认",
		}
	}

	return Signal{Target: "AVL", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确信号"}
}

// 在主函数中调用
func GetAVLSignal(klines []Kline) Signal {
	// AVL适合日线或更长周期分析

	// 1. AVL的周期选择：
	// o 日线AVL：反映中长期市场平均成本，适合趋势判断
	// o 4小时AVL：反映中短期成本变化，适合波段交易
	// o 1小时AVL：反映短期成本支撑阻力，适合日内交易

	// 2. AVL与其他指标的协同：
	// o 成交量确认：AVL突破必须配合放量才有效
	// o 均线系统：AVL + EMA20 + EMA50 形成多重确认
	// o MACD/RSI：确认动量和超买超卖状态

	// 3. AVL的独特洞察：
	// o 市场情绪：价格在AVL之上，市场乐观；价格在AVL之下，市场悲观
	// o 机构行为：AVL的陡峭变化往往反映大资金的集中建仓或出货
	// o 支撑阻力强度：AVL作为成本线，其支撑阻力效果比技术位更强烈

	// 4. 风险管理要点：
	// o 基于AVL开多：止损设在AVL下方2-3%
	// o 基于AVL开空：止损设在AVL上方2-3%
	// o 仓位管理：价格远离AVL时减少仓位，靠近AVL时增加仓位

	// 5. AVL在BTC市场的特殊性：
	// o BTC波动性大，AVL的支撑阻力效果更加明显
	// o 机构投资者特别关注成本线，AVL附近往往有大量订单堆积
	// o BTC的24/7交易特性使AVL计算更连续准确

	// 6. 避免的误区：
	// o 不要在所有时间框架使用同一AVL
	// o AVL不是领先指标，而是确认指标
	// o 在极端行情中，AVL可能会被短暂突破但很快回归

	avlData := calculateAVL(klines)

	// 分析AVL信号
	signal := analyzeAVLSignal(avlData, 10)

	return signal
}
