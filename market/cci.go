package market

import (
	"math"
)

// CCIData 存储CCI数据点
type CCIData struct {
	OpenTime     int64
	ClosePrice   float64
	HighPrice    float64
	LowPrice     float64
	TypicalPrice float64
	CCI          float64
}

// CCISignal 存储CCI分析信号
type CCISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateCCI 计算顺势指标
func calculateCCI(klines []Kline, period int) []*CCIData {
	var cciData []*CCIData
	var typicalPrices []float64

	for i, kline := range klines {
		high := kline.High
		low := kline.Low
		close := kline.Close

		// 计算典型价格
		typicalPrice := (high + low + close) / 3.0
		typicalPrices = append(typicalPrices, typicalPrice)

		// 计算CCI
		if i >= period-1 {
			// 计算移动平均
			sumTP := 0.0
			for j := 0; j < period; j++ {
				sumTP += typicalPrices[i-j]
			}
			sma := sumTP / float64(period)

			// 计算平均偏差
			sumDeviation := 0.0
			for j := 0; j < period; j++ {
				deviation := math.Abs(typicalPrices[i-j] - sma)
				sumDeviation += deviation
			}
			meanDeviation := sumDeviation / float64(period)

			// 计算CCI
			cciValue := 0.0
			if meanDeviation != 0 {
				cciValue = (typicalPrice - sma) / (0.015 * meanDeviation)
			}

			cciData = append(cciData, &CCIData{
				OpenTime:     kline.OpenTime,
				ClosePrice:   close,
				HighPrice:    high,
				LowPrice:     low,
				TypicalPrice: typicalPrice,
				CCI:          cciValue,
			})
		} else {
			// 数据不足时填充空值
			cciData = append(cciData, &CCIData{
				OpenTime:     kline.OpenTime,
				ClosePrice:   close,
				HighPrice:    high,
				LowPrice:     low,
				TypicalPrice: typicalPrice,
				CCI:          0,
			})
		}
	}
	return cciData
}

// analyzeCCISignal 分析CCI数据，生成交易信号
func analyzeCCISignal(cciData []*CCIData, lookback int) Signal {
	if len(cciData) < lookback+1 {
		return Signal{Target: "CCI", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := cciData[len(cciData)-1]
	prev := cciData[len(cciData)-2]

	// 信号1: 超买超卖线穿越
	// 从下方上穿-100线 (超卖反弹)
	if prev.CCI < -100 && current.CCI >= -100 {
		return Signal{
			Target:     "CCI",
			SignalType: "bullish_oversold",
			Side:       "buy",
			Confidence: 0.8,
			Message:    "CCI从超卖区反弹！价格回归需求强烈，开多信号",
		}
	}

	// 从上方下穿+100线 (超买回落)
	if prev.CCI > 100 && current.CCI <= 100 {
		return Signal{
			Target:     "CCI",
			SignalType: "bearish_overbought",
			Side:       "sell",
			Confidence: 0.8,
			Message:    "CCI从超买区回落！价格回调压力巨大，开空信号",
		}
	}

	// 信号2: 零轴穿越
	// CCI上穿0轴
	if prev.CCI <= 0 && current.CCI > 0 {
		return Signal{
			Target:     "CCI",
			SignalType: "bullish_zero_cross",
			Side:       "buy",
			Confidence: 0.7,
			Message:    "CCI上穿零轴！市场转强，顺势开多",
		}
	}

	// CCI下穿0轴
	if prev.CCI >= 0 && current.CCI < 0 {
		return Signal{
			Target:     "CCI",
			SignalType: "bearish_zero_cross",
			Side:       "sell",
			Confidence: 0.7,
			Message:    "CCI下穿零轴！市场转弱，顺势开空",
		}
	}

	// 信号3: 极端值预警
	if current.CCI > 200 {
		return Signal{
			Target:     "CCI",
			SignalType: "extreme_overbought",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "CCI极度超买 >200，强烈回调预警",
		}
	}

	if current.CCI < -200 {
		return Signal{
			Target:     "CCI",
			SignalType: "extreme_oversold",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "CCI极度超卖 <-200，强烈反弹预警",
		}
	}

	// 信号4: 趋势线突破 (简化版)
	// 检查最近CCI的趋势
	if len(cciData) >= 5 {
		cciList := make([]float64, len(cciData))
		for i, cci := range cciData {
			cciList[i] = cci.CCI
		}
		trend, _ := LinearRegressionAnalysis(cciList)
		if trend > 0.3 && current.CCI > 0 {
			return Signal{
				Target:     "CCI",
				SignalType: "bullish_trend",
				Side:       "buy",
				Confidence: 0.7,
				Message:    "CCI保持上升趋势且在零轴上方，多头强势",
			}
		}
		if trend < -0.3 && current.CCI < 0 {
			return Signal{
				Target:     "CCI",
				SignalType: "bearish_trend",
				Side:       "sell",
				Confidence: 0.7,
				Message:    "CCI保持下降趋势且在零轴下方，空头强势",
			}
		}
	}

	// 信号5: 背离检测
	bullishDivergence := detectCCIBullishDivergence(cciData, lookback)
	bearishDivergence := detectCCIBearishDivergence(cciData, lookback)

	if bullishDivergence {
		return Signal{
			Target:     "CCI",
			SignalType: "strong_bullish_divergence",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "发现CCI底背离！价格创新低但CCI未创新低，强烈开多信号",
		}
	}

	if bearishDivergence {
		return Signal{
			Target:     "CCI",
			SignalType: "strong_bearish_divergence",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "发现CCI顶背离！价格创新高但CCI未创新高，强烈开空信号",
		}
	}

	return Signal{Target: "CCI", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确CCI信号"}
}

// detectCCIBullishDivergence 检测CCI底背离
func detectCCIBullishDivergence(cciData []*CCIData, lookback int) bool {
	if len(cciData) < lookback*2 {
		return false
	}

	// 寻找价格低点和CCI低点
	recentPriceLow, recentPriceLowIndex := findPriceLowCCI(cciData, lookback)
	recentCCILow, recentCCILowIndex := findCCILow(cciData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentCCILowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowCCI(cciData[:recentPriceLowIndex], 5)
	prevCCILow, _ := findCCILow(cciData[:recentCCILowIndex], 5)

	// 检查背离：价格创新低，但CCI低点抬高
	if recentPriceLow < prevPriceLow && recentCCILow > prevCCILow {
		return true
	}

	return false
}

// detectCCIBearishDivergence 检测CCI顶背离
func detectCCIBearishDivergence(cciData []*CCIData, lookback int) bool {
	if len(cciData) < lookback*2 {
		return false
	}

	// 寻找价格高点和CCI高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighCCI(cciData, lookback)
	recentCCIHigh, recentCCIHighIndex := findCCIHigh(cciData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentCCIHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighCCI(cciData[:recentPriceHighIndex], 5)
	prevCCIHigh, _ := findCCIHigh(cciData[:recentCCIHighIndex], 5)

	// 检查背离：价格创新高，但CCI高点降低
	if recentPriceHigh > prevPriceHigh && recentCCIHigh < prevCCIHigh {
		return true
	}

	return false
}

// 辅助函数：在CCI数据中寻找价格高点
func findPriceHighCCI(data []*CCIData, lookback int) (float64, int) {
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

// 辅助函数：在CCI数据中寻找CCI高点
func findCCIHigh(data []*CCIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].CCI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].CCI > high {
			high = data[i].CCI
			index = i
		}
	}
	return high, index
}

// 辅助函数：在CCI数据中寻找价格低点
func findPriceLowCCI(data []*CCIData, lookback int) (float64, int) {
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

// 辅助函数：在CCI数据中寻找CCI低点
func findCCILow(data []*CCIData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].CCI
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].CCI < low {
			low = data[i].CCI
			index = i
		}
	}
	return low, index
}

func GetCCISignal(klines []Kline, peroid int) Signal {
	// 1.	CCI参数优化：
	// o	标准参数：20周期（最常用）
	// o	短线交易：14周期（更敏感）
	// o	长线投资：30-40周期（更稳定）
	// o	极端行情：10周期（捕捉快速反转）

	// 2.	CCI的独特优势：
	// o	无边界特性：可以识别传统超买超卖区之外的极端行情
	// o	统计基础：基于标准差原理，具有数学严谨性
	// o	多空均衡：0轴作为天然的多空分界线
	// o	趋势识别：既能判断超买超卖，也能识别趋势强度

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线判断主要趋势方向
	// o	动量确认：用RSI或MACD确认CCI信号
	// o	价格行为：CCI信号需要K线形态确认

	// 4.	市场环境适应：
	// o	趋势市：CCI在±100之外运行，应顺势而为
	// o	震荡市：CCI在±100之间摆动，可反向操作
	// o	突破市：CCI背离是高质量的突破前预警

	// 5.	风险管理要点：
	// o	极端值交易：在CCI>200或<-200时，止损要设置更宽
	// o	背离交易：止损设在背离起点之外
	// o	仓位管理：CCI在极端区域时减少仓位，在正常区域时正常仓位

	// 6.	BTC市场的特殊应用：
	// o	BTC波动性大，CCI经常出现极端值
	// o	关注CCI在关键支撑阻力位的表现
	// o	配合成交量确认CCI突破的有效性

	// 7.	避免的误区：
	// o	不要在强趋势中逆CCI信号交易
	// o	不要忽视0轴的多空分界意义
	// o	不要在所有市场环境下使用同一套规则

	// CCI适合4小时或日线分析
	// 计算CCI，通常使用20周期
	cciData := calculateCCI(klines, peroid)

	// 分析CCI信号
	signal := analyzeCCISignal(cciData, 20)

	return signal
}
