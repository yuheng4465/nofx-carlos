package market

// StochRSIData 存储StochRSI数据点
type StochRSIData struct {
	OpenTime       int64
	ClosePrice     float64
	RSI            float64
	StochRSI       float64
	StochRSISignal float64
	FastK          float64
	FastD          float64
}

// StochRSISignal 存储StochRSI分析信号
type StochRSISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateRSI 计算RSI指标
func calculateRSIList(prices []float64, period int) []float64 {
	rsi := make([]float64, len(prices))
	gains := make([]float64, len(prices))
	losses := make([]float64, len(prices))

	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}

	for i := period; i < len(prices); i++ {
		avgGain := 0.0
		avgLoss := 0.0

		for j := 0; j < period; j++ {
			avgGain += gains[i-j]
			avgLoss += losses[i-j]
		}

		avgGain /= float64(period)
		avgLoss /= float64(period)

		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	return rsi
}

// calculateStochRSI 计算随机相对强弱指数
func calculateStochRSI(klines []Kline, rsiPeriod int, stochPeriod int, smoothK int, smoothD int) []*StochRSIData {
	var stochRSIData []*StochRSIData
	var closes []float64

	// 提取收盘价
	for _, kline := range klines {
		closePrice := kline.Close
		closes = append(closes, closePrice)
	}

	// 计算RSI
	rsiValues := calculateRSIList(closes, rsiPeriod)

	for i := range klines {
		closePrice := klines[i].Close

		if i < rsiPeriod+stochPeriod-1 {
			// 数据不足时填充空值
			stochRSIData = append(stochRSIData, &StochRSIData{
				OpenTime:   klines[i].OpenTime,
				ClosePrice: closePrice,
				RSI:        0,
				StochRSI:   0,
				FastK:      0,
				FastD:      0,
			})
			continue
		}

		// 计算StochRSI
		lowestRSI := rsiValues[i]
		highestRSI := rsiValues[i]

		for j := 0; j < stochPeriod; j++ {
			rsiVal := rsiValues[i-j]
			if rsiVal < lowestRSI {
				lowestRSI = rsiVal
			}
			if rsiVal > highestRSI {
				highestRSI = rsiVal
			}
		}

		stochRSI := 0.0
		if highestRSI != lowestRSI {
			stochRSI = (rsiValues[i] - lowestRSI) / (highestRSI - lowestRSI) * 100
		}

		// 计算FastK和FastD
		fastK := stochRSI
		fastD := 0.0

		if i >= rsiPeriod+stochPeriod+smoothK-2 {
			// 计算FastK的平滑
			sumK := 0.0
			for j := 0; j < smoothK; j++ {
				sumK += stochRSIData[i-j].StochRSI
			}
			fastK = sumK / float64(smoothK)
		}

		if i >= rsiPeriod+stochPeriod+smoothK+smoothD-3 {
			// 计算FastD的平滑
			sumD := 0.0
			for j := 0; j < smoothD; j++ {
				sumD += stochRSIData[i-j].FastK
			}
			fastD = sumD / float64(smoothD)
		}

		stochRSIData = append(stochRSIData, &StochRSIData{
			OpenTime:       klines[i].OpenTime,
			ClosePrice:     closePrice,
			RSI:            rsiValues[i],
			StochRSI:       stochRSI,
			StochRSISignal: fastD,
			FastK:          fastK,
			FastD:          fastD,
		})
	}
	return stochRSIData
}

// analyzeStochRSISignal 分析StochRSI数据，生成交易信号
func analyzeStochRSISignal(stochRSIData []*StochRSIData, lookback int) Signal {
	if len(stochRSIData) < lookback+1 {
		return Signal{Target: "StochRSI", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := stochRSIData[len(stochRSIData)-1]
	prev := stochRSIData[len(stochRSIData)-2]

	// 信号1: 金叉死叉判断 (使用FastK和FastD)
	// 金叉: FastK上穿FastD
	if prev.FastK <= prev.FastD && current.FastK > current.FastD {
		// 在超卖区的金叉最可靠
		if current.FastK < 20 && current.FastD < 20 {
			return Signal{
				Target:     "StochRSI",
				SignalType: "strong_bullish_cross",
				Side:       "buy",
				Confidence: 0.9,
				Message:    "StochRSI在超卖区金叉！强烈开多信号",
			}
		}
		return Signal{
			Target:     "StochRSI",
			SignalType: "bullish_cross",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "StochRSI金叉，潜在开多信号",
		}
	}

	// 死叉: FastK下穿FastD
	if prev.FastK >= prev.FastD && current.FastK < current.FastD {
		// 在超买区的死叉最可靠
		if current.FastK > 80 && current.FastD > 80 {
			return Signal{
				Target:     "StochRSI",
				SignalType: "strong_bearish_cross",
				Side:       "sell",
				Confidence: 0.9,
				Message:    "StochRSI在超买区死叉！强烈开空信号",
			}
		}
		return Signal{
			Target:     "StochRSI",
			SignalType: "bearish_cross",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "StochRSI死叉，潜在开空信号",
		}
	}

	// 信号2: 超买超卖区域
	if current.FastK < 10 && current.FastD < 10 {
		return Signal{
			Target:     "StochRSI",
			SignalType: "extreme_oversold",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "StochRSI极度超卖 <10，强烈反弹预期",
		}
	}

	if current.FastK > 90 && current.FastD > 90 {
		return Signal{
			Target:     "StochRSI",
			SignalType: "extreme_overbought",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "StochRSI极度超买 >90，强烈回调预期",
		}
	}

	if current.FastK < 20 && current.FastD < 20 {
		return Signal{
			Target:     "StochRSI",
			SignalType: "oversold_zone",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "StochRSI进入超卖区，关注做多机会",
		}
	}

	if current.FastK > 80 && current.FastD > 80 {
		return Signal{
			Target:     "StochRSI",
			SignalType: "overbought_zone",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "StochRSI进入超买区，关注做空机会",
		}
	}

	// 信号3: 背离检测
	bullishDivergence := detectStochRSIBullishDivergence(stochRSIData, lookback)
	bearishDivergence := detectStochRSIBearishDivergence(stochRSIData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "StochRSI",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现StochRSI底背离！价格创新低但动量未创新低，强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "StochRSI",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现StochRSI顶背离！价格创新高但动量未创新高，强烈开空信号",
		}
	}

	// 信号4: 趋势强度
	if current.FastK > 50 && current.FastD > 50 && current.FastK > current.FastD {
		return Signal{
			Target:     "StochRSI",
			SignalType: "bullish_momentum",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "StochRSI在强势区且金叉，多头动量良好",
		}
	}

	if current.FastK < 50 && current.FastD < 50 && current.FastK < current.FastD {
		return Signal{
			Target:     "StochRSI",
			SignalType: "bearish_momentum",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "StochRSI在弱势区且死叉，空头动量良好",
		}
	}

	return Signal{Target: "StochRSI", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确StochRSI信号"}
}

// detectStochRSIBullishDivergence 检测StochRSI底背离
func detectStochRSIBullishDivergence(stochRSIData []*StochRSIData, lookback int) bool {
	if len(stochRSIData) < lookback*2 {
		return false
	}

	// 寻找价格低点和StochRSI低点（使用FastK）
	recentPriceLow, recentPriceLowIndex := findPriceLowStochRSI(stochRSIData, lookback)
	recentStochRSILow, recentStochRSILowIndex := findStochRSILow(stochRSIData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentStochRSILowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowStochRSI(stochRSIData[:recentPriceLowIndex], 5)
	prevStochRSILow, _ := findStochRSILow(stochRSIData[:recentStochRSILowIndex], 5)

	// 检查背离：价格创新低，但StochRSI低点抬高
	if recentPriceLow < prevPriceLow && recentStochRSILow > prevStochRSILow {
		return true
	}

	return false
}

// detectStochRSIBearishDivergence 检测StochRSI顶背离
func detectStochRSIBearishDivergence(stochRSIData []*StochRSIData, lookback int) bool {
	if len(stochRSIData) < lookback*2 {
		return false
	}

	// 寻找价格高点和StochRSI高点（使用FastK）
	recentPriceHigh, recentPriceHighIndex := findPriceHighStochRSI(stochRSIData, lookback)
	recentStochRSIHigh, recentStochRSIHighIndex := findStochRSIHigh(stochRSIData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentStochRSIHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighStochRSI(stochRSIData[:recentPriceHighIndex], 5)
	prevStochRSIHigh, _ := findStochRSIHigh(stochRSIData[:recentStochRSIHighIndex], 5)

	// 检查背离：价格创新高，但StochRSI高点降低
	if recentPriceHigh > prevPriceHigh && recentStochRSIHigh < prevStochRSIHigh {
		return true
	}

	return false
}

// 辅助函数：在StochRSI数据中寻找价格高点
func findPriceHighStochRSI(data []*StochRSIData, lookback int) (float64, int) {
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

// 辅助函数：在StochRSI数据中寻找StochRSI高点
func findStochRSIHigh(data []*StochRSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].FastK
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].FastK > high {
			high = data[i].FastK
			index = i
		}
	}
	return high, index
}

// 辅助函数：在StochRSI数据中寻找价格低点
func findPriceLowStochRSI(data []*StochRSIData, lookback int) (float64, int) {
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

// 辅助函数：在StochRSI数据中寻找StochRSI低点
func findStochRSILow(data []*StochRSIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].FastK
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].FastK < low {
			low = data[i].FastK
			index = i
		}
	}
	return low, index
}

func GetStochRSISignal(klines []Kline) Signal {
	// 1.	StochRSI参数优化：
	// o	标准参数：RSI(14), Stoch(14), K(3), D(3)
	// o	敏感参数：RSI(9), Stoch(9), K(2), D(2) - 更适合短线
	// o	稳定参数：RSI(21), Stoch(21), K(3), D(3) - 更适合长线

	// 2.	StochRSI的独特优势：
	// o	双重平滑：比单一RSI更少假信号
	// o	极端敏感：对动量转折点识别更早
	// o	明确区域：超买超卖区域非常清晰
	// o	动量确认：提供"动量的动量"确认

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线判断主要趋势方向
	// o	成交量：StochRSI信号配合放量更可靠
	// o	价格位置：在关键支撑阻力位的StochRSI信号更有效

	// 4.	市场环境适应：
	// o	震荡市：StochRSI的超买超卖信号极其有效
	// o	趋势市：可能在超买/超卖区停留，应结合趋势指标
	// o	突破市：StochRSI背离是高质量的突破前预警

	// 5.	风险管理要点：
	// o	超卖区开多：止损设在超卖期间的低点下方
	// o	超买区开空：止损设在超买期间的高点上方
	// o	仓位管理：在极端区域轻仓，正常区域正常仓位

	// 6.	BTC市场的特殊应用：
	// o	BTC波动性大，StochRSI经常出现极端值
	// o	关注StochRSI在关键时间周期（如4小时）的表现
	// o	配合市场情绪指标使用效果更好

	// 7.	避免的误区：
	// o	不要在强趋势中盲目逆StochRSI信号交易
	// o	不要忽视主要趋势的方向
	// o	不要在所有时间框架使用同一参数

	// StochRSI适合4小时分析
	stochRSIData := calculateStochRSI(klines, 14, 14, 3, 3)

	// 分析StochRSI信号
	signal := analyzeStochRSISignal(stochRSIData, 20)

	return signal
}
