package market

// MFIData 存储MFI数据点
type MFIData struct {
	OpenTime     int64
	ClosePrice   float64
	HighPrice    float64
	LowPrice     float64
	Volume       float64
	TypicalPrice float64
	RawMoneyFlow float64
	MFI          float64
	PositiveFlow float64 // 正资金流
	NegativeFlow float64 // 负资金流
}

// MFISignal 存储MFI分析信号
type MFISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateMFI 计算资金流量指数
func calculateMFI(klines []Kline, period int) []*MFIData {
	var mfiData []*MFIData
	var typicalPrices []float64
	var rawMoneyFlows []float64

	for i, kline := range klines {
		high := kline.High
		low := kline.Low
		close := kline.Close
		volume := kline.Volume

		// 计算典型价格和原始资金流
		typicalPrice := (high + low + close) / 3.0
		rawMoneyFlow := typicalPrice * volume

		typicalPrices = append(typicalPrices, typicalPrice)
		rawMoneyFlows = append(rawMoneyFlows, rawMoneyFlow)

		// 计算MFI
		if i >= period {
			positiveFlow := 0.0
			negativeFlow := 0.0

			for j := 0; j < period; j++ {
				idx := i - j
				prevIdx := i - j - 1

				if typicalPrices[idx] > typicalPrices[prevIdx] {
					positiveFlow += rawMoneyFlows[idx]
				} else if typicalPrices[idx] < typicalPrices[prevIdx] {
					negativeFlow += rawMoneyFlows[idx]
				}
				// 如果相等，则不计算
			}

			mfiValue := 0.0
			if negativeFlow != 0 {
				moneyRatio := positiveFlow / negativeFlow
				mfiValue = 100 - (100 / (1 + moneyRatio))
			} else if positiveFlow > 0 {
				mfiValue = 100 // 全部为正资金流
			} else {
				mfiValue = 0 // 全部为负资金流或持平
			}

			mfiData = append(mfiData, &MFIData{
				OpenTime:     kline.OpenTime,
				ClosePrice:   close,
				HighPrice:    high,
				LowPrice:     low,
				Volume:       volume,
				TypicalPrice: typicalPrice,
				RawMoneyFlow: rawMoneyFlow,
				MFI:          mfiValue,
				PositiveFlow: positiveFlow,
				NegativeFlow: negativeFlow,
			})
		} else {
			// 数据不足时填充空值
			mfiData = append(mfiData, &MFIData{
				OpenTime:     kline.OpenTime,
				ClosePrice:   close,
				HighPrice:    high,
				LowPrice:     low,
				Volume:       volume,
				TypicalPrice: typicalPrice,
				RawMoneyFlow: rawMoneyFlow,
				MFI:          0,
			})
		}
	}
	return mfiData
}

// analyzeMFISignal 分析MFI数据，生成交易信号
func analyzeMFISignal(mfiData []*MFIData, lookback int) Signal {
	if len(mfiData) < lookback+1 {
		return Signal{Target: "MFI", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := mfiData[len(mfiData)-1]
	prev := mfiData[len(mfiData)-2]

	// 信号1: 超买超卖线穿越
	// 从下方上穿20线 (超卖反弹)
	if prev.MFI < 20 && current.MFI >= 20 {
		return Signal{
			Target:     "MFI",
			SignalType: "bullish_oversold",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "MFI从超卖区反弹！资金开始流入，强烈开多信号",
		}
	}

	// 从上方下穿80线 (超买回落)
	if prev.MFI > 80 && current.MFI <= 80 {
		return Signal{
			Target:     "MFI",
			SignalType: "bearish_overbought",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "MFI从超买区回落！资金开始流出，强烈开空信号",
		}
	}

	// 信号2: 中轴线穿越
	// 在强势区域上穿50
	if current.MFI > 30 && current.MFI < 70 {
		if prev.MFI < 50 && current.MFI >= 50 {
			return Signal{
				Target:     "MFI",
				SignalType: "bullish_momentum",
				Side:       "buy",
				Confidence: 0.7,
				Message:    "MFI上穿50中轴，资金流入占据主导",
			}
		}
	}

	// 在弱势区域下穿50
	if current.MFI > 30 && current.MFI < 70 {
		if prev.MFI > 50 && current.MFI <= 50 {
			return Signal{
				Target:     "MFI",
				SignalType: "bearish_momentum",
				Side:       "sell",
				Confidence: 0.7,
				Message:    "MFI下穿50中轴，资金流出占据主导",
			}
		}
	}

	// 信号3: 背离检测
	bullishDivergence := detectMFIBullishDivergence(mfiData, lookback)
	bearishDivergence := detectMFIBearishDivergence(mfiData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "MFI",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现MFI底背离！价格创新低但资金未流出，强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "MFI",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现MFI顶背离！价格创新高但资金未流入，强烈开空信号",
		}
	}

	// 信号4: 极端值警告
	if current.MFI > 90 {
		return Signal{
			Target:     "MFI",
			SignalType: "extreme_overbought",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "MFI极度超买 >90，警惕大幅回调",
		}
	}

	if current.MFI < 10 {
		return Signal{
			Target:     "MFI",
			SignalType: "extreme_oversold",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "MFI极度超卖 <10，关注强势反弹",
		}
	}

	return Signal{Target: "MFI", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确MFI信号"}
}

// detectMFIBullishDivergence 检测MFI底背离
func detectMFIBullishDivergence(mfiData []*MFIData, lookback int) bool {
	if len(mfiData) < lookback*2 {
		return false
	}

	// 寻找价格低点和MFI低点
	recentPriceLow, recentPriceLowIndex := findPriceLowMFI(mfiData, lookback)
	recentMFILow, recentMFILowIndex := findMFILow(mfiData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentMFILowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowMFI(mfiData[:recentPriceLowIndex], 5)
	prevMFILow, _ := findMFILow(mfiData[:recentMFILowIndex], 5)

	// 检查背离：价格创新低，但MFI低点抬高
	if recentPriceLow < prevPriceLow && recentMFILow > prevMFILow {
		return true
	}

	return false
}

// detectMFIBearishDivergence 检测MFI顶背离
func detectMFIBearishDivergence(mfiData []*MFIData, lookback int) bool {
	if len(mfiData) < lookback*2 {
		return false
	}

	// 寻找价格高点和MFI高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighMFI(mfiData, lookback)
	recentMFIHigh, recentMFIHighIndex := findMFIHigh(mfiData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentMFIHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighMFI(mfiData[:recentPriceHighIndex], 5)
	prevMFIHigh, _ := findMFIHigh(mfiData[:recentMFIHighIndex], 5)

	// 检查背离：价格创新高，但MFI高点降低
	if recentPriceHigh > prevPriceHigh && recentMFIHigh < prevMFIHigh {
		return true
	}

	return false
}

// 辅助函数：在MFI数据中寻找价格高点
func findPriceHighMFI(data []*MFIData, lookback int) (float64, int) {
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

// 辅助函数：在MFI数据中寻找MFI高点
func findMFIHigh(data []*MFIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].MFI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].MFI > high {
			high = data[i].MFI
			index = i
		}
	}
	return high, index
}

// 辅助函数：在MFI数据中寻找价格低点
func findPriceLowMFI(data []*MFIData, lookback int) (float64, int) {
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

// 辅助函数：在MFI数据中寻找MFI低点
func findMFILow(data []*MFIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].MFI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].MFI < low {
			low = data[i].MFI
			index = i
		}
	}
	return low, index
}

// 类似的 findPriceHighMFI, findMFIHigh 函数...

func GetMFISignal(klines []Kline, peroid int) Signal {
	// 1.	MFI参数优化：
	// o	标准参数：14周期（平衡敏感度和稳定性）
	// o	短线交易：9-12周期（更敏感）
	// o	长线投资：20-25周期（更稳定）

	// 2.	MFI的独特优势：
	// o	量价确认：比RSI多了成交量的维度
	// o	资金洞察：直接反映大资金的动向
	// o	背离质量：MFI背离的信号质量通常高于RSI

	// 3.	与其他指标配合：
	// o	价格行为：MFI信号需要价格走势确认
	// o	趋势指标：结合均线或MACD判断主要趋势方向
	// o	波动率指标：用布林带识别价格位置

	// 4.	市场环境适应：
	// o	趋势市：MFI的超买超卖信号更可靠
	// o	震荡市：MFI可能在中间区域徘徊，信号不明确
	// o	突破市：MFI背离是高质量的突破前预警信号

	// 5.	风险管理要点：
	// o	MFI超卖开多：止损设在超卖期间的低点下方
	// o	MFI超买开空：止损设在超买期间的高点上方
	// o	背离交易：止损设在背离起点之外

	// 6.	BTC市场的特殊应用：
	// o	BTC成交量巨大，MFI的资金流分析特别有效
	// o	关注MFI的极端值（<10或>90），往往对应重要转折点
	// o	配合4小时或日线使用，避免短线噪音

	// 7.	避免的误区：
	// o	不要单独依赖MFI超买超卖信号逆势交易
	// o	在强趋势中，MFI可能在超买/超卖区停留很长时间
	// o	不要忽视成交量的确认作用

	// MFI适合4小时或日线分析
	// 计算MFI，通常使用14周期
	mfiData := calculateMFI(klines, peroid)

	// 分析MFI信号
	signal := analyzeMFISignal(mfiData, 20)

	return signal
}
