package market

// EMVData 存储EMV数据点
type EMVData struct {
	OpenTime     int64
	ClosePrice   float64
	HighPrice    float64
	LowPrice     float64
	Volume       float64
	MidPrice     float64 // 中间价
	PriceChange  float64 // 价格变化
	BoxRatio     float64 // 盒比率
	EMV          float64 // EMV值
	EMVMovingAvg float64 // EMV移动平均（信号线）
}

// calculateEMV 计算简易波动指标
func calculateEMV(klines []Kline, period int, movingAvgPeriod int) []*EMVData {
	var emvData []*EMVData
	var emvValues []float64

	for i, kline := range klines {
		high := kline.High
		low := kline.Low
		close := kline.Close
		volume := kline.Volume

		// 计算中间价
		midPrice := (high + low) / 2.0

		// 计算价格变化
		priceChange := 0.0
		if i > 0 {
			prevMidPrice := emvData[i-1].MidPrice
			priceChange = midPrice - prevMidPrice
		}

		// 计算盒比率
		boxRatio := 0.0
		priceRange := high - low
		if priceRange != 0 {
			// 成交量除以10000是为了调整数值规模
			boxRatio = (volume / 10000.0) / priceRange
		}

		// 计算EMV
		emvValue := 0.0
		if boxRatio != 0 {
			emvValue = priceChange / boxRatio
		}

		emvValues = append(emvValues, emvValue)

		// 计算EMV移动平均（信号线）
		emvMovingAvg := 0.0
		if i >= movingAvgPeriod-1 {
			sumEMV := 0.0
			for j := 0; j < movingAvgPeriod; j++ {
				sumEMV += emvValues[i-j]
			}
			emvMovingAvg = sumEMV / float64(movingAvgPeriod)
		}

		emvData = append(emvData, &EMVData{
			OpenTime:     kline.OpenTime,
			ClosePrice:   close,
			HighPrice:    high,
			LowPrice:     low,
			Volume:       volume,
			MidPrice:     midPrice,
			PriceChange:  priceChange,
			BoxRatio:     boxRatio,
			EMV:          emvValue,
			EMVMovingAvg: emvMovingAvg,
		})
	}
	return emvData
}

// analyzeEMVSignal 分析EMV数据，生成交易信号
func analyzeEMVSignal(emvData []*EMVData, lookback int) Signal {
	if len(emvData) < lookback+1 {
		return Signal{Target: "EMV", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := emvData[len(emvData)-1]
	prev := emvData[len(emvData)-2]

	// 信号1: 零轴穿越
	// EMV上穿零轴
	if prev.EMV <= 0 && current.EMV > 0 {
		return Signal{
			Target:     "EMV",
			SignalType: "bullish_zero_cross",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "EMV上穿零轴！资金开始有效推动价格上涨，开多信号",
		}
	}

	// EMV下穿零轴
	if prev.EMV >= 0 && current.EMV < 0 {
		return Signal{
			Target:     "EMV",
			SignalType: "bearish_zero_cross",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "EMV下穿零轴！资金开始有效推动价格下跌，开空信号",
		}
	}

	// 信号2: 信号线交叉
	// EMV金叉（上穿信号线）
	if prev.EMV <= prev.EMVMovingAvg && current.EMV > current.EMVMovingAvg {
		if current.EMV > 0 {
			return Signal{
				Target:     "EMV",
				SignalType: "bullish_golden_cross",
				Side:       "buy",
				Confidence: 0.7,
				Message:    "EMV金叉且在零轴上方！资金效率加速向上，开多信号",
			}
		} else {
			return Signal{
				Target:     "EMV",
				SignalType: "potential_bullish_cross",
				Side:       "buy",
				Confidence: 0.6,
				Message:    "EMV金叉但在零轴下方，可能反弹但需谨慎",
			}
		}
	}

	// EMV死叉（下穿信号线）
	if prev.EMV >= prev.EMVMovingAvg && current.EMV < current.EMVMovingAvg {
		if current.EMV < 0 {
			return Signal{
				Target:     "EMV",
				SignalType: "bearish_dead_cross",
				Side:       "sell",
				Confidence: 0.7,
				Message:    "EMV死叉且在零轴下方！资金效率加速向下，开空信号",
			}
		} else {
			return Signal{
				Target:     "EMV",
				SignalType: "potential_bearish_cross",
				Side:       "sell",
				Confidence: 0.6,
				Message:    "EMV死叉但在零轴上方，可能回调但需谨慎",
			}
		}
	}

	// 信号3: 资金效率强度
	if current.EMV > 0 && current.EMV > prev.EMV {
		return Signal{
			Target:     "EMV",
			SignalType: "bullish_efficiency",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "EMV为正且加速上升，资金推动效率提高",
		}
	}

	if current.EMV < 0 && current.EMV < prev.EMV {
		return Signal{
			Target:     "EMV",
			SignalType: "bearish_efficiency",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "EMV为负且加速下降，资金推动效率提高",
		}
	}

	// 信号4: 背离检测
	bullishDivergence := detectEMVBullishDivergence(emvData, lookback)
	bearishDivergence := detectEMVBearishDivergence(emvData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "EMV",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现EMV底背离！价格创新低但资金效率未创新低，强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "EMV",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现EMV顶背离！价格创新高但资金效率未创新高，强烈开空信号",
		}
	}

	// 信号5: 效率衰减预警
	if current.EMV > 0 && current.EMV < prev.EMV {
		return Signal{
			Target:     "EMV",
			SignalType: "bullish_efficiency_decay",
			Side:       "none",
			Confidence: 0.5,
			Message:    "EMV为正但开始衰减，资金推动效率下降",
		}
	}

	if current.EMV < 0 && current.EMV > prev.EMV {
		return Signal{
			Target:     "EMV",
			SignalType: "bearish_efficiency_decay",
			Side:       "none",
			Confidence: 0.5,
			Message:    "EMV为负但开始衰减，资金推动效率下降",
		}
	}

	return Signal{Target: "EMV", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确EMV信号"}
}

// detectEMVBullishDivergence 检测EMV底背离
func detectEMVBullishDivergence(emvData []*EMVData, lookback int) bool {
	if len(emvData) < lookback*2 {
		return false
	}

	// 寻找价格低点和EMV低点
	recentPriceLow, recentPriceLowIndex := findPriceLowEMV(emvData, lookback)
	recentEMVLow, recentEMVLowIndex := findEMVLow(emvData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentEMVLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowEMV(emvData[:recentPriceLowIndex], 5)
	prevEMVLow, _ := findEMVLow(emvData[:recentEMVLowIndex], 5)

	// 检查背离：价格创新低，但EMV低点抬高
	if recentPriceLow < prevPriceLow && recentEMVLow > prevEMVLow {
		return true
	}

	return false
}

// detectEMVBearishDivergence 检测EMV顶背离
func detectEMVBearishDivergence(emvData []*EMVData, lookback int) bool {
	if len(emvData) < lookback*2 {
		return false
	}

	// 寻找价格高点和EMV高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighEMV(emvData, lookback)
	recentEMVHigh, recentEMVHighIndex := findEMVHigh(emvData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentEMVHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighEMV(emvData[:recentPriceHighIndex], 5)
	prevEMVHigh, _ := findEMVHigh(emvData[:recentEMVHighIndex], 5)

	// 检查背离：价格创新高，但EMV高点降低
	if recentPriceHigh > prevPriceHigh && recentEMVHigh < prevEMVHigh {
		return true
	}

	return false
}

// 辅助函数：在EMV数据中寻找价格高点
func findPriceHighEMV(data []*EMVData, lookback int) (float64, int) {
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

// 辅助函数：在EMV数据中寻找EMV高点
func findEMVHigh(data []*EMVData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].EMV
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].EMV > high {
			high = data[i].EMV
			index = i
		}
	}
	return high, index
}

// 辅助函数：在EMV数据中寻找价格低点
func findPriceLowEMV(data []*EMVData, lookback int) (float64, int) {
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

// 辅助函数：在EMV数据中寻找EMV低点
func findEMVLow(data []*EMVData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].EMV
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].EMV < low {
			low = data[i].EMV
			index = i
		}
	}
	return low, index
}

func GetEMVSignal(klines []Kline, peroid int, movingAvgPeriod int) Signal {
	// 1.	EMV参数优化：
	// o	标准参数：EMV(14), 信号线(9)
	// o	短线交易：EMV(9), 信号线(6) - 更敏感
	// o	长线投资：EMV(21), 信号线(14) - 更稳定

	// 2.	EMV的独特优势：
	// o	量价结合：同时考虑价格和成交量的变化
	// o	效率衡量：衡量单位成交量推动价格变化的效率
	// o	资金意图：反映大资金的真实意图和操作效率
	// o	突破确认：有效识别真假突破

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线判断主要趋势方向
	// o	成交量指标：用OBV或MFI确认资金流向
	// o	动量指标：用RSI或MACD确认动量状态

	// 4.	市场环境适应：
	// o	趋势市：EMV的零轴穿越信号非常有效
	// o	突破市：EMV能有效识别突破的真实性
	// o	震荡市：EMV可能在零轴附近徘徊，信号不明确

	// 5.	风险管理要点：
	// o	零轴穿越：在EMV穿越零轴时入场
	// o	背离交易：止损设置在背离起点之外
	// o	效率衰减：当EMV开始衰减时考虑减仓

	// 6.	BTC市场的特殊应用：
	// o	BTC成交量巨大，EMV效果很好
	// o	关注EMV在关键价格位的表现
	// o	BTC的机构资金动向在EMV上反映明显

	// 7.	避免的误区：
	// o	不要忽视成交量的配合确认
	// o	不要在EMV极端值时盲目交易
	// o	不要单独使用EMV，要结合趋势分析

	// EMV适合4小时分析
	// 计算EMV，标准参数(14, 9)
	emvData := calculateEMV(klines, peroid, movingAvgPeriod)

	// 分析EMV信号
	signal := analyzeEMVSignal(emvData, 20)

	return signal
}
