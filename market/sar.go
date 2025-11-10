package market

// SARData 存储SAR数据点
type SARData struct {
	OpenTime   int64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	SAR        float64 // SAR值
	Trend      string  // 趋势方向
	AF         float64 // 加速因子
	EP         float64 // 极值点
}

// SARSignal 存储SAR分析信号
type SARSignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// calculateSAR 计算抛物线转向指标
func calculateSAR(klines []Kline, initialAF float64, maxAF float64, step float64) []*SARData {
	var sarData []*SARData

	if len(klines) == 0 {
		return sarData
	}

	// 初始化第一个SAR点
	firstHigh := klines[0].High
	firstLow := klines[0].Low

	// 假设初始趋势为下跌
	trend := "down"
	sar := firstHigh // 从最高点开始
	ep := firstLow   // 极值点为最低点
	af := initialAF

	for i, kline := range klines {
		high := kline.High
		low := kline.Low
		close := kline.Close

		if i == 0 {
			sarData = append(sarData, &SARData{
				OpenTime:   kline.OpenTime,
				ClosePrice: close,
				HighPrice:  high,
				LowPrice:   low,
				SAR:        sar,
				Trend:      trend,
				AF:         af,
				EP:         ep,
			})
			continue
		}

		prev := sarData[i-1]

		// 根据前一个趋势计算当前SAR
		var currentSAR float64
		if prev.Trend == "up" {
			// 上升趋势中的SAR计算
			currentSAR = prev.SAR + prev.AF*(prev.EP-prev.SAR)

			// 检查是否转向
			if low <= currentSAR {
				// 转向下跌
				trend = "down"
				currentSAR = prev.EP // 从极值点开始
				ep = high
				af = initialAF
			} else {
				// 继续上升趋势
				trend = "up"
				if high > prev.EP {
					ep = high
					af = min(prev.AF+step, maxAF)
				} else {
					ep = prev.EP
					af = prev.AF
				}
			}
		} else {
			// 下跌趋势中的SAR计算
			currentSAR = prev.SAR - prev.AF*(prev.SAR-prev.EP)

			// 检查是否转向
			if high >= currentSAR {
				// 转向上升
				trend = "up"
				currentSAR = prev.EP // 从极值点开始
				ep = low
				af = initialAF
			} else {
				// 继续下跌趋势
				trend = "down"
				if low < prev.EP {
					ep = low
					af = min(prev.AF+step, maxAF)
				} else {
					ep = prev.EP
					af = prev.AF
				}
			}
		}

		sarData = append(sarData, &SARData{
			OpenTime:   kline.OpenTime,
			ClosePrice: close,
			HighPrice:  high,
			LowPrice:   low,
			SAR:        currentSAR,
			Trend:      trend,
			AF:         af,
			EP:         ep,
		})
	}
	return sarData
}

// min 返回最小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// 转换为字符串
func getSARDataString(sarData []*SARData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(sarData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := sarData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.SAR)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// analyzeSARSignal 分析SAR数据，生成交易信号
func analyzeSARSignal(sarData []*SARData, lookback int, period string) *Signal {
	if len(sarData) < lookback+1 {
		return &Signal{Target: TargetSAR, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := sarData[len(sarData)-1]
	prev := sarData[len(sarData)-2]

	// 信号1: 趋势反转信号
	if prev.Trend != current.Trend {
		if current.Trend == "up" {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "bullish_reversal",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.9,
				Message:    "SAR趋势转多！价格突破SAR点，强烈开多信号",
			}
		} else {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "bearish_reversal",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.9,
				Message:    "SAR趋势转空！价格跌破SAR点，强烈开空信号",
			}
		}
	}

	// 信号2: 趋势持续信号
	if current.Trend == "up" {
		// 检查上升趋势的强度
		distancePercent := (current.ClosePrice - current.SAR) / current.SAR * 100

		if distancePercent > 5.0 {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "strong_bullish_trend",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.8,
				Message:    "价格远离SAR点，上升趋势强劲",
			}
		}

		// 检查AF加速因子
		if current.AF > 0.05 {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "bullish_acceleration",
				Side:       SideBuy,
				Period:     period,
				Confidence: 0.7,
				Message:    "SAR加速上行，上涨动量增强",
			}
		}

		return &Signal{
			Target:     TargetSAR,
			SignalType: "bullish_trend",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "上升趋势持续，持有多头",
		}
	} else {
		// 下降趋势
		distancePercent := (current.SAR - current.ClosePrice) / current.SAR * 100

		if distancePercent > 5.0 {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "strong_bearish_trend",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.8,
				Message:    "价格远离SAR点，下跌趋势强劲",
			}
		}

		if current.AF > 0.05 {
			return &Signal{
				Target:     TargetSAR,
				SignalType: "bearish_acceleration",
				Side:       SideSell,
				Period:     period,
				Confidence: 0.7,
				Message:    "SAR加速下行，下跌动量增强",
			}
		}

		return &Signal{
			Target:     TargetSAR,
			SignalType: "bearish_trend",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "下降趋势持续，持有空头",
		}
	}
}

func GetSARSignal(klines []Kline, period string) *Signal {
	// 1.	SAR参数优化：
	// o	标准参数：AF=0.02, Max AF=0.2, Step=0.02
	// o	敏感参数：AF=0.01, Max AF=0.1, Step=0.01（更适合短线）
	// o	稳定参数：AF=0.03, Max AF=0.3, Step=0.03（更适合长线）

	// 2.	SAR的独特优势：
	// o	自动止损：SAR点就是天然止损位
	// o	趋势跟踪：让利润奔跑，及时止损
	// o	简单明了：视觉上非常直观
	// o	反转及时：在趋势转折时快速发出信号

	// 3.	市场环境适应：
	// o	趋势市：SAR表现极佳，信号准确
	// o	震荡市：SAR会产生较多假信号，应减少使用或配合其他指标

	// 4.	与其他指标配合：
	// o	ADX：用ADX>25来确认趋势强度，过滤震荡市信号
	// o	均线系统：SAR方向与均线排列一致时信号更可靠
	// o	成交量：SAR反转时配合放量，信号更可信

	// 5.	风险管理要点：
	// o	止损设置：多单止损设在SAR点下方，空单止损设在SAR点上方
	// o	移动止损：随着趋势发展，不断将止损移动至最新的SAR点
	// o	仓位管理：在趋势初期轻仓，趋势确认后加仓

	// 6.	BTC市场的特殊应用：
	// o	BTC波动大，SAR的止损功能尤其重要
	// o	在BTC强趋势行情中，SAR能捕捉大部分利润
	// o	配合4小时或日线使用，避免短线噪音干扰

	// 7.	避免的误区：
	// o	不要在震荡市中盲目跟随SAR信号
	// o	不要逆SAR趋势交易
	// o	不要忽视SAR的移动止损功能

	// SAR适合4小时或日线分析
	// 计算SAR，标准参数(0.02, 0.2, 0.02)
	sarData := calculateSAR(klines, 0.02, 0.2, 0.02)

	// 分析SAR信号
	signal := analyzeSARSignal(sarData, 10, period)

	return signal
}
