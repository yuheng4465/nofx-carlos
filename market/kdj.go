package market

// KDJData 存储KDJ数据点
type KDJData struct {
	OpenTime   int64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	KValue     float64
	DValue     float64
	JValue     float64
	RSV        float64
}

// KDJSignal 存储KDJ分析信号
type KDJSignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateKDJ 计算KDJ指标
func calculateKDJ(klines []Kline, period int, kSmooth int, dSmooth int) []*KDJData {
	var kdjData []*KDJData

	for i := range klines {
		if i < period-1 {
			// 数据不足时填充空值
			closePrice := klines[i].Close
			kdjData = append(kdjData, &KDJData{
				OpenTime:   klines[i].OpenTime,
				ClosePrice: closePrice,
				KValue:     50, // 默认中间值
				DValue:     50,
				JValue:     50,
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

		// 计算RSV
		rsv := 0.0
		if periodHigh != periodLow {
			rsv = (closePrice - periodLow) / (periodHigh - periodLow) * 100
		}

		// 计算K、D、J值
		var kValue, dValue, jValue float64

		if i == period-1 {
			// 第一个计算点
			kValue = 50
			dValue = 50
		} else {
			// 使用前一日值计算
			prev := kdjData[i-1]
			kValue = (prev.KValue*float64(kSmooth-1) + rsv) / float64(kSmooth)
			dValue = (prev.DValue*float64(dSmooth-1) + kValue) / float64(dSmooth)
		}

		jValue = 3*kValue - 2*dValue

		kdjData = append(kdjData, &KDJData{
			OpenTime:   klines[i].OpenTime,
			ClosePrice: closePrice,
			HighPrice:  periodHigh,
			LowPrice:   periodLow,
			KValue:     kValue,
			DValue:     dValue,
			JValue:     jValue,
			RSV:        rsv,
		})
	}
	return kdjData
}

// analyzeKDJSignal 分析KDJ数据，生成交易信号
func analyzeKDJSignal(kdjData []*KDJData, lookback int) Signal {
	if len(kdjData) < lookback+1 {
		return Signal{Target: "KDJ", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := kdjData[len(kdjData)-1]
	prev := kdjData[len(kdjData)-2]

	// 信号1: 金叉死叉判断
	// 金叉: K线上穿D线
	if prev.KValue <= prev.DValue && current.KValue > current.DValue {
		// 在超卖区的金叉最可靠
		if current.KValue < 20 && current.DValue < 20 {
			return Signal{
				Target:     "KDJ",
				SignalType: "strong_bullish_cross",
				Side:       "buy",
				Confidence: 0.9,
				Message:    "KDJ在超卖区金叉！强烈开多信号",
			}
		}
		return Signal{
			Target:     "KDJ",
			SignalType: "bullish_cross",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "KDJ金叉，潜在开多信号",
		}
	}

	// 死叉: K线下穿D线
	if prev.KValue >= prev.DValue && current.KValue < current.DValue {
		// 在超买区的死叉最可靠
		if current.KValue > 80 && current.DValue > 80 {
			return Signal{
				Target:     "KDJ",
				SignalType: "strong_bearish_cross",
				Side:       "sell",
				Confidence: 0.9,
				Message:    "KDJ在超买区死叉！强烈开空信号",
			}
		}
		return Signal{
			Target:     "KDJ",
			SignalType: "bearish_cross",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "KDJ死叉，潜在开空信号",
		}
	}

	// 信号2: J线极端值
	if current.JValue < 0 {
		return Signal{
			Target:     "KDJ",
			SignalType: "extreme_oversold",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "J线极度超卖 <0，强烈反弹预期",
		}
	}

	if current.JValue > 100 {
		return Signal{
			Target:     "KDJ",
			SignalType: "extreme_overbought",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "J线极度超买 >100，强烈回调预期",
		}
	}

	// 信号3: 超买超卖区域
	if current.KValue < 20 && current.DValue < 20 {
		return Signal{
			Target:     "KDJ",
			SignalType: "oversold_zone",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "KDJ进入超卖区，关注做多机会",
		}
	}

	if current.KValue > 80 && current.DValue > 80 {
		return Signal{
			Target:     "KDJ",
			SignalType: "overbought_zone",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "KDJ进入超买区，关注做空机会",
		}
	}

	// 信号4: 背离检测
	bullishDivergence := detectKDJBullishDivergence(kdjData, lookback)
	bearishDivergence := detectKDJBearishDivergence(kdjData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "KDJ",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现KDJ底背离！价格创新低但KDJ未创新低，强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "KDJ",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现KDJ顶背离！价格创新高但KDJ未创新高，强烈开空信号",
		}
	}

	// 信号5: 趋势强度
	if current.KValue > current.DValue && current.DValue > current.JValue && current.KValue > 50 {
		return Signal{
			Target:     "KDJ",
			SignalType: "strong_bullish_trend",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "K>D>J且在50上方，强势多头格局",
		}
	}

	if current.KValue < current.DValue && current.DValue < current.JValue && current.KValue < 50 {
		return Signal{
			Target:     "KDJ",
			SignalType: "strong_bearish_trend",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "K<D<J且在50下方，强势空头格局",
		}
	}

	return Signal{Target: "KDJ", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确KDJ信号"}
}

// detectKDJBullishDivergence 检测KDJ底背离
func detectKDJBullishDivergence(kdjData []*KDJData, lookback int) bool {
	if len(kdjData) < lookback*2 {
		return false
	}

	// 寻找价格低点和KDJ低点（使用K值）
	recentPriceLow, recentPriceLowIndex := findPriceLowKDJ(kdjData, lookback)
	recentKDJLow, recentKDJLowIndex := findKDJLow(kdjData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentKDJLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowKDJ(kdjData[:recentPriceLowIndex], 5)
	prevKDJLow, _ := findKDJLow(kdjData[:recentKDJLowIndex], 5)

	// 检查背离：价格创新低，但KDJ低点抬高
	if recentPriceLow < prevPriceLow && recentKDJLow > prevKDJLow {
		return true
	}

	return false
}

// detectKDJBearishDivergence 检测KDJ顶背离
func detectKDJBearishDivergence(kdjData []*KDJData, lookback int) bool {
	if len(kdjData) < lookback*2 {
		return false
	}

	// 寻找价格高点和KDJ高点（使用K值）
	recentPriceHigh, recentPriceHighIndex := findPriceHighKDJ(kdjData, lookback)
	recentKDJHigh, recentKDJHighIndex := findKDJHigh(kdjData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentKDJHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighKDJ(kdjData[:recentPriceHighIndex], 5)
	prevKDJHigh, _ := findKDJHigh(kdjData[:recentKDJHighIndex], 5)

	// 检查背离：价格创新高，但KDJ高点降低
	if recentPriceHigh > prevPriceHigh && recentKDJHigh < prevKDJHigh {
		return true
	}

	return false
}

// 辅助函数：在KDJ数据中寻找价格高点
func findPriceHighKDJ(data []*KDJData, lookback int) (float64, int) {
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

// 辅助函数：在KDJ数据中寻找KDJ高点
func findKDJHigh(data []*KDJData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].KValue
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].KValue > high {
			high = data[i].KValue
			index = i
		}
	}
	return high, index
}

// 辅助函数：在KDJ数据中寻找价格低点
func findPriceLowKDJ(data []*KDJData, lookback int) (float64, int) {
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

// 辅助函数：在KDJ数据中寻找KDJ低点
func findKDJLow(data []*KDJData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].KValue
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].KValue < low {
			low = data[i].KValue
			index = i
		}
	}
	return low, index
}

// 在主函数中调用
func GetKDJSignal(klines []Kline, period int, kSmooth int, dSmooth int) Signal {
	// 1.	KDJ参数优化：
	// o	标准参数：9,3,3（平衡敏感度和稳定性）
	// o	短线交易：6,3,3或9,2,2（更敏感）
	// o	长线投资：14,3,3或21,3,3（更稳定）

	// 2.	KDJ的独特优势：
	// o	三重确认：K、D、J三线提供多重信号过滤
	// o	灵敏度选择：J线最敏感，D线最稳定，K线居中
	// o	极端预警：J线的极端值提供早期反转预警

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线或MACD判断主要趋势方向
	// o	成交量：KDJ金叉死叉配合放量更可靠
	// o	布林带：识别价格位置和波动性

	// 4.	市场环境适应：
	// o	震荡市：KDJ的超买超卖信号非常有效
	// o	趋势市：KDJ可能在超买/超卖区停留，应顺势而为
	// o	突破市：KDJ背离是高质量的突破前预警

	// 5.	风险管理要点：
	// o	超卖区开多：止损设在超卖期间的低点下方
	// o	超买区开空：止损设在超买期间的高点上方
	// o	背离交易：止损设在背离起点之外

	// 6.	BTC市场的特殊应用：
	// o	BTC波动性大，KDJ的J线经常出现极端值
	// o	关注KDJ在关键支撑阻力位的表现
	// o	配合4小时或日线使用，避免短线噪音

	// 7.	避免的误区：
	// o	不要在强趋势中逆KDJ超买超卖信号交易
	// o	不要忽视三线的排列关系和趋势背景
	// o	不要在所有时间框架使用同一参数

	// KDJ适合4小时或日线分析
	kdjData := calculateKDJ(klines, period, kSmooth, dSmooth)

	// 分析KDJ信号
	signal := analyzeKDJSignal(kdjData, 20)

	return signal
}
