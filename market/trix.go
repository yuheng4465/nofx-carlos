package market

// TRIXData 存储TRIX数据点
type TRIXData struct {
	OpenTime   int64
	ClosePrice float64
	EMA1       float64 // 第一次EMA
	EMA2       float64 // 第二次EMA
	EMA3       float64 // 第三次EMA
	TRIX       float64 // TRIX值
	TRIXSignal float64 // TRIX信号线
	Histogram  float64 // 柱状图
}

// TRIXSignal 存储TRIX分析信号
type TRIXSignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateEMA 计算指数移动平均
func calculateEMATRIX(prices []float64, period int) []float64 {
	ema := make([]float64, len(prices))
	multiplier := 2.0 / (float64(period) + 1.0)

	for i, price := range prices {
		if i == 0 {
			ema[i] = price
		} else {
			ema[i] = (price-ema[i-1])*multiplier + ema[i-1]
		}
	}
	return ema
}

// calculateTRIX 计算TRIX指标
func calculateTRIX(klines []Kline, period int) []*TRIXData {
	var trixData []*TRIXData

	var closes []float64

	for _, kline := range klines {
		closePrice := kline.Close
		closes = append(closes, closePrice)
	}

	// 计算三次EMA
	ema1 := calculateEMATRIX(closes, period)
	ema2 := calculateEMATRIX(ema1, period)
	ema3 := calculateEMATRIX(ema2, period)

	// 计算TRIX值和信号线
	for i := range ema3 {
		var trixValue float64
		var trixSignal float64
		var histogram float64

		if i > 0 {
			// 计算TRIX百分比变化
			if ema3[i-1] != 0 {
				trixValue = (ema3[i] - ema3[i-1]) / ema3[i-1] * 100
			}

			// 计算TRIX信号线
			if i == 1 {
				trixSignal = trixValue
			} else {
				// 信号线是TRIX的EMA
				signalMultiplier := 2.0 / (float64(period) + 1.0)
				trixSignal = (trixValue-trixData[i-1].TRIXSignal)*signalMultiplier + trixData[i-1].TRIXSignal
			}

			histogram = trixValue - trixSignal
		}

		closePrice := klines[i].Close
		trixData = append(trixData, &TRIXData{
			OpenTime:   klines[i].OpenTime,
			ClosePrice: closePrice,
			EMA1:       ema1[i],
			EMA2:       ema2[i],
			EMA3:       ema3[i],
			TRIX:       trixValue,
			TRIXSignal: trixSignal,
			Histogram:  histogram,
		})
	}
	return trixData
}

// analyzeTRIXSignal 分析TRIX数据，生成交易信号
func analyzeTRIXSignal(trixData []*TRIXData, lookback int) Signal {
	if len(trixData) < lookback+1 {
		return Signal{Target: "TRIX", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := trixData[len(trixData)-1]
	prev := trixData[len(trixData)-2]

	// 信号1: 零轴穿越
	// TRIX上穿零轴
	if prev.TRIX <= 0 && current.TRIX > 0 {
		return Signal{
			Target:     "TRIX",
			SignalType: "bullish_zero_cross",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "TRIX上穿零轴！多头动量确立，强烈开多信号",
		}
	}

	// TRIX下穿零轴
	if prev.TRIX >= 0 && current.TRIX < 0 {
		return Signal{
			Target:     "TRIX",
			SignalType: "bearish_zero_cross",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "TRIX下穿零轴！空头动量确立，强烈开空信号",
		}
	}

	// 信号2: 金叉死叉
	// TRIX金叉
	if prev.TRIX <= prev.TRIXSignal && current.TRIX > current.TRIXSignal {
		if current.TRIX > 0 {
			return Signal{
				Target:     "TRIX",
				SignalType: "bullish_golden_cross",
				Side:       "buy",
				Confidence: 0.8,
				Message:    "TRIX在零轴上金叉！上涨动量加速",
			}
		} else {
			return Signal{
				Target:     "TRIX",
				SignalType: "potential_bullish_cross",
				Side:       "buy",
				Confidence: 0.6,
				Message:    "TRIX在零轴下金叉，可能反弹但需谨慎",
			}
		}
	}

	// TRIX死叉
	if prev.TRIX >= prev.TRIXSignal && current.TRIX < current.TRIXSignal {
		if current.TRIX < 0 {
			return Signal{
				Target:     "TRIX",
				SignalType: "bearish_dead_cross",
				Side:       "sell",
				Confidence: 0.8,
				Message:    "TRIX在零轴下死叉！下跌动量加速",
			}
		} else {
			return Signal{
				Target:     "TRIX",
				SignalType: "potential_bearish_cross",
				Side:       "sell",
				Confidence: 0.6,
				Message:    "TRIX在零轴上死叉，可能回调但需谨慎",
			}
		}
	}

	// 信号3: 柱状图动量
	if current.Histogram > 0 && current.Histogram > prev.Histogram {
		return Signal{
			Target:     "TRIX",
			SignalType: "bullish_momentum",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "TRIX柱状线向上放大，上涨动量增强",
		}
	}

	if current.Histogram < 0 && current.Histogram < prev.Histogram {
		return Signal{
			Target:     "TRIX",
			SignalType: "bearish_momentum",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "TRIX柱状线向下放大，下跌动量增强",
		}
	}

	// 信号4: 背离检测
	bullishDivergence := detectTRIXBullishDivergence(trixData, lookback)
	bearishDivergence := detectTRIXBearishDivergence(trixData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "TRIX",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现TRIX底背离！强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "TRIX",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现TRIX顶背离！强烈开空信号",
		}
	}

	return Signal{Target: "TRIX", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确TRIX信号"}
}

// detectTRIXBullishDivergence 检测TRIX底背离
func detectTRIXBullishDivergence(trixData []*TRIXData, lookback int) bool {
	if len(trixData) < lookback*2 {
		return false
	}

	// 寻找价格低点和TRIX低点
	recentPriceLow, recentPriceLowIndex := findPriceLowTRIX(trixData, lookback)
	recentTRIXLow, recentTRIXLowIndex := findTRIXLow(trixData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentTRIXLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowTRIX(trixData[:recentPriceLowIndex], 5)
	prevTRIXLow, _ := findTRIXLow(trixData[:recentTRIXLowIndex], 5)

	// 检查背离：价格创新低，但TRIX低点抬高
	if recentPriceLow < prevPriceLow && recentTRIXLow > prevTRIXLow {
		return true
	}

	return false
}

// detectTRIXBearishDivergence 检测TRIX顶背离
func detectTRIXBearishDivergence(trixData []*TRIXData, lookback int) bool {
	if len(trixData) < lookback*2 {
		return false
	}

	// 寻找价格高点和TRIX高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighTRIX(trixData, lookback)
	recentTRIXHigh, recentTRIXHighIndex := findTRIXHigh(trixData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentTRIXHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighTRIX(trixData[:recentPriceHighIndex], 5)
	prevTRIXHigh, _ := findTRIXHigh(trixData[:recentTRIXHighIndex], 5)

	// 检查背离：价格创新高，但TRIX高点降低
	if recentPriceHigh > prevPriceHigh && recentTRIXHigh < prevTRIXHigh {
		return true
	}

	return false
}

// 辅助函数：在TRIX数据中寻找价格高点
func findPriceHighTRIX(data []*TRIXData, lookback int) (float64, int) {
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

// 辅助函数：在TRIX数据中寻找TRIX高点
func findTRIXHigh(data []*TRIXData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].TRIX
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].TRIX > high {
			high = data[i].TRIX
			index = i
		}
	}
	return high, index
}

// 辅助函数：在TRIX数据中寻找价格低点
func findPriceLowTRIX(data []*TRIXData, lookback int) (float64, int) {
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

// 辅助函数：在TRIX数据中寻找TRIX低点
func findTRIXLow(data []*TRIXData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].TRIX
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].TRIX < low {
			low = data[i].TRIX
			index = i
		}
	}
	return low, index
}

func GetTRIXSignal(klines []Kline, peroid int) Signal {
	// 1.	TRIX参数优化：
	// o	短线交易：使用9-12周期，更敏感
	// o	中线交易：使用15-20周期，平衡敏感度与稳定性
	// o	长线投资：使用25-30周期，更稳定

	// 2.	TRIX的独特优势：
	// o	领先指标：TRIX的变化通常领先于价格趋势变化
	// o	噪音过滤：三重平滑有效消除市场杂讯
	// o	趋势质量：TRIX的稳定性反映趋势的健康程度

	// 3.	与其他指标配合：
	// o	RSI：确认超买超卖状态
	// o	成交量：确认TRIX信号的有效性
	// o	布林带：识别价格位置和波动性

	// 4.	多时间框架分析：
	// o	日线TRIX：判断主要趋势方向
	// o	4小时TRIX：寻找交易机会
	// o	1小时TRIX：精确定位入场点

	// 5.	风险管理要点：
	// o	TRIX零轴穿越：止损设在穿越前的高低点
	// o	TRIX背离交易：止损设在背离起点之外
	// o	仓位控制：TRIX在零轴附近时轻仓，远离零轴时正常仓位

	// 6.	避免的误区：
	// o	不要在所有市场环境下使用同一TRIX参数
	// o	TRIX在震荡市中效果较差，应配合趋势过滤
	// o	不要忽视TRIX柱状图的早期警告信号

	// TRIX适合4小时或日线分析
	// 计算TRIX，通常使用15周期
	trixData := calculateTRIX(klines, 15)

	// 分析TRIX信号
	signal := analyzeTRIXSignal(trixData, 20)

	return signal
}
