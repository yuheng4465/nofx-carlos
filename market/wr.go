package market

// WRData 存储WR数据点
type WRData struct {
	OpenTime   int64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	WR         float64
}

// WRSignal 存储WR分析信号
type WRSignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateWR 计算威廉姆斯指标
func calculateWR(klines []Kline, period int) []*WRData {
	var wrData []*WRData

	for i := range klines {
		if i < period-1 {
			// 数据不足时填充空值
			closePrice := klines[i].Close
			wrData = append(wrData, &WRData{
				OpenTime:   klines[i].OpenTime,
				ClosePrice: closePrice,
				WR:         0,
			})
			continue
		}

		// 计算周期内的最高价和最低价
		periodHigh := 0.0
		periodLow := 0.0

		for j := 0; j < period; j++ {
			high := klines[i-j].High
			low := klines[i-j].Low

			if j == 0 {
				periodHigh = high
				periodLow = low
			} else {
				if high > periodHigh {
					periodHigh = high
				}
				if low < periodLow {
					periodLow = low
				}
			}
		}

		closePrice := klines[i].Close

		// 计算WR
		wrValue := 0.0
		if periodHigh != periodLow {
			wrValue = (periodHigh - closePrice) / (periodHigh - periodLow) * (-100)
		}

		wrData = append(wrData, &WRData{
			OpenTime:   klines[i].OpenTime,
			ClosePrice: closePrice,
			HighPrice:  periodHigh,
			LowPrice:   periodLow,
			WR:         wrValue,
		})
	}
	return wrData
}

// analyzeWRSignal 分析WR数据，生成交易信号
func analyzeWRSignal(wrData []*WRData, lookback int) Signal {
	if len(wrData) < lookback+1 {
		return Signal{Target: "WR", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := wrData[len(wrData)-1]
	prev := wrData[len(wrData)-2]

	// 信号1: 超买超卖线穿越
	// 从超卖区上穿-80线 (反弹信号)
	if prev.WR < -80 && current.WR >= -80 {
		return Signal{
			Target:     "WR",
			SignalType: "bullish_oversold",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "WR从超卖区反弹！价格回归需求强烈，开多信号",
		}
	}

	// 从超买区下穿-20线 (回落信号)
	if prev.WR > -20 && current.WR <= -20 {
		return Signal{
			Target:     "WR",
			SignalType: "bearish_overbought",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "WR从超买区回落！价格回调压力巨大，开空信号",
		}
	}

	// 信号2: 中轴线穿越
	// WR上穿-50线
	if prev.WR <= -50 && current.WR > -50 {
		return Signal{
			Target:     "WR",
			SignalType: "bullish_momentum",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "WR上穿中轴线-50！市场转强，顺势开多",
		}
	}

	// WR下穿-50线
	if prev.WR >= -50 && current.WR < -50 {
		return Signal{
			Target:     "WR",
			SignalType: "bearish_momentum",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "WR下穿中轴线-50！市场转弱，顺势开空",
		}
	}

	// 信号3: 极端值预警
	if current.WR > -10 {
		return Signal{
			Target:     "WR",
			SignalType: "extreme_overbought",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "WR极度超买 >-10，强烈回调预警",
		}
	}

	if current.WR < -90 {
		return Signal{
			Target:     "WR",
			SignalType: "extreme_oversold",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "WR极度超卖 <-90，强烈反弹预警",
		}
	}

	// 信号4: 趋势判断
	// WR在超卖区突破自身下降趋势线（做多）
	wrList := make([]float64, len(wrData))
	for i, vwap := range wrData {
		wrList[i] = vwap.WR
	}
	slope, _ := LinearRegressionAnalysis(wrList)
	if slope < 0 && current.WR > prev.WR {
		return Signal{
			Target:     "WR",
			SignalType: "bullish_bias",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "WR在强势区域，多头占优",
		}
	}
	// WR在超卖区跌破自身上升趋势线（做多）
	if slope > 0 && current.WR < prev.WR {
		return Signal{
			Target:     "WR",
			SignalType: "bearish_bias",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "WR在弱势区域，空头占优",
		}
	}

	// 信号5: 背离检测
	bullishDivergence := detectWRBullishDivergence(wrData, lookback)
	if bullishDivergence {
		return Signal{
			Target:     "WR",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现WR底背离！价格创新低但WR未创新低，强烈开多信号",
		}
	}

	bearishDivergence := detectWRBearishDivergence(wrData, lookback)
	if bearishDivergence {
		return Signal{
			Target:     "WR",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现WR顶背离！价格创新高但WR未创新高，强烈开空信号",
		}
	}

	return Signal{Target: "WR", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确WR信号"}
}

// detectWRBullishDivergence 检测WR底背离
func detectWRBullishDivergence(wrData []*WRData, lookback int) bool {
	if len(wrData) < lookback*2 {
		return false
	}

	// 寻找价格低点和WR低点
	recentPriceLow, recentPriceLowIndex := findPriceLowWR(wrData, lookback)
	recentWRLow, recentWRLowIndex := findWRLow(wrData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentWRLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowWR(wrData[:recentPriceLowIndex], 5)
	prevWRLow, _ := findWRLow(wrData[:recentWRLowIndex], 5)

	// 检查背离：价格创新低，但WR低点抬高（WR值更大）
	if recentPriceLow < prevPriceLow && recentWRLow > prevWRLow {
		return true
	}

	return false
}

// detectWRBearishDivergence 检测WR顶背离
func detectWRBearishDivergence(wrData []*WRData, lookback int) bool {
	if len(wrData) < lookback*2 {
		return false
	}

	// 寻找价格高点和WR高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighWR(wrData, lookback)
	recentWRHigh, recentWRHighIndex := findWRHigh(wrData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentWRHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighWR(wrData[:recentPriceHighIndex], 5)
	prevWRHigh, _ := findWRHigh(wrData[:recentWRHighIndex], 5)

	// 检查背离：价格创新高，但WR高点降低（WR值更小）
	if recentPriceHigh > prevPriceHigh && recentWRHigh < prevWRHigh {
		return true
	}

	return false
}

// 辅助函数：在WR数据中寻找价格低点
func findPriceHighWR(data []*WRData, lookback int) (float64, int) {
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

// 辅助函数：在WR数据中寻找价格低点
func findPriceLowWR(data []*WRData, lookback int) (float64, int) {
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

// 辅助函数：在WR数据中寻找WR低点（WR值更小）
func findWRLow(data []*WRData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].WR
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].WR < low {
			low = data[i].WR
			index = i
		}
	}
	return low, index
}

// 辅助函数：在WR数据中寻找WR高点（WR值更大）
func findWRHigh(data []*WRData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].WR
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].WR > high {
			high = data[i].WR
			index = i
		}
	}
	return high, index
}

func GetWRSignal(klines []Kline) Signal {
	// 1.	WR参数优化：
	// o	标准参数：14周期（最常用）
	// o	短线交易：7-10周期（更敏感）
	// o	长线投资：20-30周期（更稳定）
	// o	双WR组合：使用不同周期的WR（如7和14）进行确认

	// 2.	WR的独特优势：
	// o	反向视角：提供与其他指标不同的市场视角
	// o	极端敏感：对价格区间的相对位置反应迅速
	// o	明确边界：超买超卖区域非常清晰明确
	// o	简单直观：计算简单，易于理解

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线判断主要趋势方向
	// o	动量确认：用RSI或MACD确认WR信号
	// o	成交量：WR信号配合放量更可靠

	// 4.	市场环境适应：
	// o	震荡市：WR的超买超卖信号极其有效
	// o	趋势市：WR可能在超买/超卖区停留，应结合趋势
	// o	突破市：WR背离是高质量的突破前预警

	// 5.	风险管理要点：
	// o	WR超卖开多：止损设在超卖期间的低点下方
	// o	WR超买开空：止损设在超买期间的高点上方
	// o	仓位管理：在WR极端区域轻仓，正常区域正常仓位

	// 6.	BTC市场的特殊应用：
	// o	BTC波动性大，WR经常出现极端值
	// o	关注WR在关键支撑阻力位的表现
	// o	配合市场情绪指标使用效果更好

	// 7.	避免的误区：
	// o	注意方向：记住WR是反向指标，高位是超买，低位是超卖
	// o	不要逆势：在强趋势中不要盲目逆WR信号交易
	// o	多重确认：不要单独依赖WR信号进行交易

	// WR适合4小时或日线分析
	// 计算WR，通常使用14周期
	wrData := calculateWR(klines, 14)

	// 分析WR信号
	signal := analyzeWRSignal(wrData, 20)

	return signal
}
