package market

import (
	"fmt"
	"math"
)

// DMIData 存储DMI数据点
type DMIData struct {
	OpenTime   int64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	PlusDI     float64 // +DI
	MinusDI    float64 // -DI
	ADX        float64 // 平均趋向指数
	TR         float64 // 真实波幅
	PlusDM     float64 // +DM
	MinusDM    float64 // -DM
}

// calculateDMI 计算动向指标
func calculateDMI(klines []Kline, period int) []*DMIData {
	var dmiData []*DMIData
	var trValues []float64
	var plusDMValues []float64
	var minusDMValues []float64
	// 收盘价、最高价、最低价
	// var closes, highs, lows []float64
	// for _, kline := range klines {
	// 	closes = append(closes, kline.Close)
	// 	highs = append(highs, kline.High)
	// 	lows = append(lows, kline.Low)
	// }

	// MinusDIData := talib.MinusDI(highs, lows, closes, period)
	// PlusDMData := talib.PlusDI(highs, lows, closes, period)
	// AdxData := talib.Adx(highs, lows, closes, period)

	// for i := range period {
	// 	dmiData = append(dmiData, &DMIData{
	// 		PlusDI:  PlusDMData[i],
	// 		MinusDI: MinusDIData[i],
	// 		ADX:     AdxData[i],
	// 	})
	// }

	// return dmiData

	for i, kline := range klines {
		high := kline.High
		low := kline.Low
		close := kline.Close

		// 计算真实波幅(TR)
		var tr float64
		if i == 0 {
			tr = high - low
		} else {
			prevClose := klines[i-1].Close
			method1 := high - low
			method2 := math.Abs(high - prevClose)
			method3 := math.Abs(low - prevClose)
			tr = math.Max(method1, math.Max(method2, method3))
		}

		// 计算方向运动(+DM和-DM)
		var plusDM, minusDM float64
		if i > 0 {
			prevHigh := klines[i-1].High
			prevLow := klines[i-1].Low

			upMove := high - prevHigh
			downMove := prevLow - low

			if upMove > downMove && upMove > 0 {
				plusDM = upMove
			} else {
				plusDM = 0
			}

			if downMove > upMove && downMove > 0 {
				minusDM = downMove
			} else {
				minusDM = 0
			}
		}

		trValues = append(trValues, tr)
		plusDMValues = append(plusDMValues, plusDM)
		minusDMValues = append(minusDMValues, minusDM)

		// 计算平滑后的值
		if i >= period-1 {
			// 计算初始平均值
			sumTR := 0.0
			sumPlusDM := 0.0
			sumMinusDM := 0.0

			for j := 0; j < period; j++ {
				sumTR += trValues[i-j]
				sumPlusDM += plusDMValues[i-j]
				sumMinusDM += minusDMValues[i-j]
			}

			atr := sumTR / float64(period)
			plusDIAvg := sumPlusDM / float64(period)
			minusDIAvg := sumMinusDM / float64(period)

			// 计算+DI和-DI
			plusDI := 0.0
			minusDI := 0.0
			if atr != 0 {
				plusDI = (plusDIAvg / atr) * 100
				minusDI = (minusDIAvg / atr) * 100
			}

			// 计算DX和ADX
			dx := 0.0
			if plusDI+minusDI != 0 {
				dx = (math.Abs(plusDI-minusDI) / (plusDI + minusDI)) * 100
			}

			// 计算ADX（第一个ADX值是DX的简单平均）
			adx := 0.0
			if i == period-1 {
				adx = dx
			} else if i >= period {
				prevADX := dmiData[i-1].ADX
				adx = (prevADX*float64(period-1) + dx) / float64(period)
			}

			dmiData = append(dmiData, &DMIData{
				OpenTime:   kline.OpenTime,
				ClosePrice: close,
				HighPrice:  high,
				LowPrice:   low,
				PlusDI:     plusDI,
				MinusDI:    minusDI,
				ADX:        adx,
				TR:         tr,
				PlusDM:     plusDM,
				MinusDM:    minusDM,
			})
		} else {
			// 数据不足时填充空值
			dmiData = append(dmiData, &DMIData{
				OpenTime:   kline.OpenTime,
				ClosePrice: close,
				HighPrice:  high,
				LowPrice:   low,
				PlusDI:     0,
				MinusDI:    0,
				ADX:        0,
			})
		}
	}
	return dmiData
}

// 转换为字符串
func getDMIDataString(dmiData []*DMIData, period int) string {
	var plusDI, minusDI, ADX []float64

	// 取尾部数据
	startIndex := len(dmiData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := dmiData[startIndex:]
	for _, v := range lastData {
		plusDI = append(plusDI, v.PlusDI)
		minusDI = append(minusDI, v.MinusDI)
		ADX = append(ADX, v.ADX)
	}

	// 将字节切片转换为字符串
	plusDIStr := formatFloatSlice(plusDI)
	minusDIStr := formatFloatSlice(minusDI)
	ADXStr := formatFloatSlice(ADX)

	return fmt.Sprintf("PlusDI: %s , MinusDI: %s , ADX: %s", plusDIStr, minusDIStr, ADXStr)
}

// analyzeDMISignal 分析DMI数据，生成交易信号
func analyzeDMISignal(dmiData []*DMIData, lookback int, period string) *Signal {
	if len(dmiData) < lookback+1 {
		return &Signal{Target: TargetDMI, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := dmiData[len(dmiData)-1]
	prev := dmiData[len(dmiData)-2]

	// 信号1: DI线交叉
	// +DI上穿-DI（金叉）
	if prev.PlusDI <= prev.MinusDI && current.PlusDI > current.MinusDI {
		if current.ADX > 25 {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "strong_bullish_cross",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.9,
				Message:    "+DI上穿-DI且ADX>25！强烈上升趋势确认，开多信号",
			}
		}
		return &Signal{
			Target:     TargetDMI,
			SignalType: "bullish_cross",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.7,
			Message:    "+DI上穿-DI，潜在上升趋势，关注做多机会",
		}
	}

	// -DI上穿+DI（死叉）
	if prev.MinusDI <= prev.PlusDI && current.MinusDI > current.PlusDI {
		if current.ADX > 25 {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "strong_bearish_cross",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.9,
				Message:    "-DI上穿+DI且ADX>25！强烈下降趋势确认，开空信号",
			}
		}
		return &Signal{
			Target:     TargetDMI,
			SignalType: "bearish_cross",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.7,
			Message:    "-DI上穿+DI，潜在下降趋势，关注做空机会",
		}
	}

	// 信号2: 趋势强度判断
	if current.PlusDI > current.MinusDI && current.ADX > 30 {
		return &Signal{
			Target:     TargetDMI,
			SignalType: "strong_uptrend",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.8,
			Message:    "强劲上升趋势！+DI > -DI 且 ADX > 30",
		}
	}

	if current.MinusDI > current.PlusDI && current.ADX > 30 {
		return &Signal{
			Target:     TargetDMI,
			SignalType: "strong_downtrend",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.8,
			Message:    "强劲下降趋势！-DI > +DI 且 ADX > 30",
		}
	}

	// 信号3: ADX趋势启动
	if current.ADX > 20 && prev.ADX <= 20 {
		if current.PlusDI > current.MinusDI {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "uptrend_start",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.7,
				Message:    "ADX突破20！上升趋势启动，关注做多机会",
			}
		} else if current.MinusDI > current.PlusDI {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "downtrend_start",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.7,
				Message:    "ADX突破20！下降趋势启动，关注做空机会",
			}
		}
	}

	// 信号4: 极端趋势预警
	if current.ADX > 50 {
		return &Signal{
			Target:     TargetDMI,
			SignalType: "extreme_trend",
			Side:       SideWaring,
			Period:     period,
			Confidence: 0.6,
			Message:    "ADX > 50，趋势可能过度延伸，警惕反转",
		}
	}

	// 信号5: 盘整市场
	if current.ADX < 20 {
		if current.PlusDI > current.MinusDI {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "weak_uptrend",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.5,
				Message:    "市场盘整，轻微多头倾向，谨慎做多",
			}
		} else if current.MinusDI > current.PlusDI {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "weak_downtrend",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.5,
				Message:    "市场盘整，轻微空头倾向，谨慎做空",
			}
		} else {
			return &Signal{
				Target:     TargetDMI,
				SignalType: "consolidation",
				Side:       SideNone,
				Period:     period,
				Confidence: 0.5,
				Message:    "市场盘整，多空力量平衡，建议观望",
			}
		}
	}

	return &Signal{Target: TargetDMI, SignalType: "none", Side: SideNone, Period: period, Confidence: 0.5, Message: "未发现明确DMI信号"}
}

// 在主函数中调用
func GetDMISignal(klines []Kline, period int, timePeriod string) *Signal {
	// 1.	DMI参数优化：
	// o	标准参数：14周期（最常用）
	// o	短线交易：7-10周期（更敏感）
	// o	长线投资：20-30周期（更稳定）
	// o	趋势过滤：ADX的周期可以与DI不同

	// 2.	DMI的独特优势：
	// o	趋势确认：同时提供趋势方向和强度信息
	// o	过滤震荡：ADX有效过滤盘整市场的假信号
	// o	多空量化：+DI和-DI精确量化多空力量对比
	// o	早期预警：ADX抬头往往早于价格突破

	// 3.	与其他指标配合：
	// o	均线系统：DMI确认趋势方向，均线提供具体入场点
	// o	MACD：DMI判断趋势，MACD判断动量
	// o	布林带：DMI判断趋势，布林带识别价格位置

	// 4.	市场环境适应：
	// o	趋势市：DMI表现极佳，应主要依靠DMI信号
	// o	震荡市：DMI效果较差，应减少使用或配合震荡指标
	// o	转折点：ADX从低位抬头是重要的趋势启动信号

	// 5.	风险管理要点：
	// o	趋势交易：在ADX>25时，止损可以设置更宽
	// o	盘整回避：在ADX<20时，减少交易或使用其他策略
	// o	仓位管理：趋势明确时（ADX>30）可以适当加大仓位

	// 6.	BTC市场的特殊应用：
	// o	BTC趋势性明显，DMI效果很好
	// o	关注4小时和日线级别的DMI信号
	// o	BTC的强趋势中，ADX可能长期维持在较高水平

	// 7.	避免的误区：
	// o	不要在ADX<20时使用趋势跟踪策略
	// o	不要忽视DI交叉与ADX的配合使用
	// o	不要在极端趋势（ADX>50）中过度追涨杀跌

	// DMI适合4小时或日线分析
	// 计算DMI，通常使用14周期
	dmiData := calculateDMI(klines, period)

	// 分析DMI信号
	signal := analyzeDMISignal(dmiData, 20, timePeriod)

	return signal
}
