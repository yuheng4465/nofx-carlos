package market

import "fmt"

// MTMData 存储MTM数据点
type MTMData struct {
	OpenTime     int64
	ClosePrice   float64
	MTM          float64 // 动量值
	MTMMovingAvg float64 // MTM移动平均（信号线）
	MomentumRate float64 // 动量变化率
	PriceChange  float64 // 价格变化
}

// calculateMTM 计算动量指标
func calculateMTM(klines []Kline, period int, signalPeriod int) []*MTMData {
	if len(klines) == 0 {
		return nil
	}

	mtmData := make([]*MTMData, len(klines))

	// 第一遍：计算基础MTM值
	for i := range klines {
		mtmData[i] = &MTMData{
			OpenTime:   klines[i].OpenTime,
			ClosePrice: klines[i].Close,
		}

		if i >= period {
			prevClose := klines[i-period].Close
			mtmData[i].MTM = klines[i].Close - prevClose

			if prevClose != 0 {
				mtmData[i].MomentumRate = (klines[i].Close/prevClose - 1) * 100
			}
		}

		if i > 0 {
			mtmData[i].PriceChange = klines[i].Close - klines[i-1].Close
		}
	}

	// 第二遍：计算移动平均（确保所有MTM值都已计算）
	for i := range mtmData {
		if i >= period+signalPeriod-1 {
			sum := 0.0
			for j := i - signalPeriod + 1; j <= i; j++ {
				sum += mtmData[j].MTM
			}
			mtmData[i].MTMMovingAvg = sum / float64(signalPeriod)
		}
	}

	return mtmData
}

// 转换为字符串
func getMTMDataString(mtmData []*MTMData, period int) string {
	var MTM, MTMMovingAvg []float64
	// 取尾部数据
	startIndex := len(mtmData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := mtmData[startIndex:]
	for _, v := range lastData {
		MTM = append(MTM, v.MTM)
		MTMMovingAvg = append(MTMMovingAvg, v.MTMMovingAvg)
	}

	// 将字节切片转换为字符串
	MTMStr := formatFloatSlice(MTM)
	MTMMovingAvgStr := formatFloatSlice(MTMMovingAvg)

	return fmt.Sprintf("MTM: %s , MTMMovingAvg: %s", MTMStr, MTMMovingAvgStr)
}

// analyzeMTMSignal 分析MTM数据，生成交易信号
func analyzeMTMSignal(mtmData []*MTMData, lookback int, period string) *Signal {
	if len(mtmData) < lookback+1 {
		return &Signal{Target: TargetMTM, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := mtmData[len(mtmData)-1]
	prev := mtmData[len(mtmData)-2]

	// 信号1: 零轴穿越
	// MTM上穿零轴
	if prev.MTM <= 0 && current.MTM > 0 {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bullish_zero_cross",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.8,
			Message:    "MTM上穿零轴！下跌动量转为上升动量，开多信号",
		}
	}

	// MTM下穿零轴
	if prev.MTM >= 0 && current.MTM < 0 {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bearish_zero_cross",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.8,
			Message:    "MTM下穿零轴！上涨动量转为下跌动量，开空信号",
		}
	}

	// 信号2: 信号线交叉
	// MTM上穿信号线
	if current.MTMMovingAvg != 0 && prev.MTM <= prev.MTMMovingAvg && current.MTM > current.MTMMovingAvg {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bullish_signal_cross",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.7,
			Message:    "MTM上穿信号线！动量加速向上，开多信号",
		}
	}

	// MTM下穿信号线
	if current.MTMMovingAvg != 0 && prev.MTM >= prev.MTMMovingAvg && current.MTM < current.MTMMovingAvg {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bearish_signal_cross",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.7,
			Message:    "MTM下穿信号线！动量加速向下，开空信号",
		}
	}

	// 信号3: 动量强度判断
	if current.MTM > 0 && current.MTM > prev.MTM {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bullish_acceleration",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "MTM为正且加速上升，上涨动量强劲",
		}
	}

	if current.MTM < 0 && current.MTM < prev.MTM {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bearish_acceleration",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "MTM为负且加速下降，下跌动量强劲",
		}
	}

	// 信号4: 动量减速预警
	if current.MTM > 0 && current.MTM < prev.MTM {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bullish_deceleration",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.5,
			Message:    "MTM为正但开始减速，上涨动量减弱",
		}
	}

	if current.MTM < 0 && current.MTM > prev.MTM {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "bearish_deceleration",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.5,
			Message:    "MTM为负但开始减速，下跌动量减弱",
		}
	}

	// 信号5: 背离检测
	bullishDivergence := detectMTMBullishDivergence(mtmData, lookback)
	bearishDivergence := detectMTMBearishDivergence(mtmData, lookback)

	if bullishDivergence {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "strong_bullish_divergence",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.9,
			Message:    "发现MTM底背离！价格创新低但动量未创新低，强烈开多信号",
		}
	}

	if bearishDivergence {
		return &Signal{
			Target:     TargetMTM,
			SignalType: "strong_bearish_divergence",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.9,
			Message:    "发现MTM顶背离！价格创新高但动量未创新高，强烈开空信号",
		}
	}

	return &Signal{Target: TargetMTM, SignalType: "none", Side: SideNone, Period: period, Confidence: 0.5, Message: "未发现明确MTM信号"}
}

// detectMTMBullishDivergence 检测MTM底背离
func detectMTMBullishDivergence(mtmData []*MTMData, lookback int) bool {
	if len(mtmData) < lookback*2 {
		return false
	}

	// 寻找价格低点和MTM低点
	recentPriceLow, recentPriceLowIndex := findPriceLowMTM(mtmData, lookback)
	recentMTMLow, recentMTMLowIndex := findMTMLow(mtmData, lookback)

	// 寻找前一个低点
	if recentPriceLowIndex < 5 || recentMTMLowIndex < 5 {
		return false
	}

	prevPriceLow, _ := findPriceLowMTM(mtmData[:recentPriceLowIndex], 5)
	prevMTMLow, _ := findMTMLow(mtmData[:recentMTMLowIndex], 5)

	// 检查背离：价格创新低，但MTM低点抬高
	if recentPriceLow < prevPriceLow && recentMTMLow > prevMTMLow {
		return true
	}

	return false
}

// detectMTMBearishDivergence 检测MTM顶背离
func detectMTMBearishDivergence(mtmData []*MTMData, lookback int) bool {
	if len(mtmData) < lookback*2 {
		return false
	}

	// 寻找价格高点和MTM高点
	recentPriceHigh, recentPriceHighIndex := findPriceHighMTM(mtmData, lookback)
	recentMTMHigh, recentMTMHighIndex := findMTMHigh(mtmData, lookback)

	// 寻找前一个高点
	if recentPriceHighIndex < 5 || recentMTMHighIndex < 5 {
		return false
	}

	prevPriceHigh, _ := findPriceHighMTM(mtmData[:recentPriceHighIndex], 5)
	prevMTMHigh, _ := findMTMHigh(mtmData[:recentMTMHighIndex], 5)

	// 检查背离：价格创新高，但MTM高点降低
	if recentPriceHigh > prevPriceHigh && recentMTMHigh < prevMTMHigh {
		return true
	}

	return false
}

// 辅助函数：在MTM数据中寻找价格高点
func findPriceHighMTM(data []*MTMData, lookback int) (float64, int) {
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

// 辅助函数：在MTM数据中寻找MTM高点
func findMTMHigh(data []*MTMData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].MTM
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].MTM > high {
			high = data[i].MTM
			index = i
		}
	}
	return high, index
}

// 辅助函数：在MTM数据中寻找价格低点
func findPriceLowMTM(data []*MTMData, lookback int) (float64, int) {
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

// 辅助函数：在MTM数据中寻找MTM低点
func findMTMLow(data []*MTMData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].MTM
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].MTM < low {
			low = data[i].MTM
			index = i
		}
	}
	return low, index
}

func GetMTMSignal(klines []Kline, period string) *Signal {
	// 1.	MTM参数优化：
	// o	标准参数：10周期（平衡敏感度和稳定性）
	// o	短线交易：6-8周期（更敏感）
	// o	长线投资：12-14周期（更稳定）
	// o	信号线：通常使用6周期移动平均

	// 2.	MTM的独特优势：
	// o	直接测量：直接反映价格变化的速度
	// o	领先指标：动量变化通常领先于价格趋势变化
	// o	加速度识别：能识别价格运动的加速和减速阶段
	// o	简单有效：计算简单但信号明确

	// 3.	与其他指标配合：
	// o	趋势指标：结合均线判断主要趋势方向
	// o	震荡指标：用RSI或StochRSI确认超买超卖状态
	// o	成交量：MTM信号配合放量更可靠

	// 4.	市场环境适应：
	// o	趋势市：MTM的零轴穿越信号非常有效
	// o	震荡市：MTM可能在零轴附近徘徊，信号不明确
	// o	转折点：MTM背离是最高质量的转折信号

	// 5.	风险管理要点：
	// o	零轴穿越：在MTM穿越零轴时入场，止损设置在近期高低点
	// o	背离交易：止损设置在背离起点之外
	// o	动量减速：当MTM开始减速时，考虑减仓或设置保护止损

	// 6.	BTC市场的特殊应用：
	// o	BTC动量效应明显，MTM效果很好
	// o	关注MTM在关键支撑阻力位的表现
	// o	BTC的强动量行情中，MTM可能持续在极端区域

	// 7.	避免的误区：
	// o	不要单独使用MTM，要结合趋势分析
	// o	不要在MTM极端值时盲目追涨杀跌
	// o	不要忽视MTM减速的预警信号

	// MTM适合4小时分析
	// 计算MTM，标准参数(10, 6)
	mtmData := calculateMTM(klines, 10, 6)

	// 分析MTM信号
	signal := analyzeMTMSignal(mtmData, 20, period)

	return signal
}
