package market

// VolumeData 存储成交量数据点
type VolumeData struct {
	OpenTime           int64
	ClosePrice         float64
	Volume             float64
	VolumeMA           float64 // 成交量移动平均线
	VolumeRatio        float64 // 成交量比率 (当前成交量 / 平均成交量)
	PriceChange        float64 // 价格变化
	PriceChangePercent float64 // 价格变化百分比
}

// calculateVolumeMA 计算成交量移动平均线
func calculateVolumeMA(klines []Kline, period int) []*VolumeData {
	var volumeData []*VolumeData

	for i, kline := range klines {
		closePrice := kline.Close
		volume := kline.Volume

		// 计算价格变化
		priceChange := 0.0
		priceChangePercent := 0.0
		if i > 0 {
			prevClose := klines[i-1].Close
			priceChange = closePrice - prevClose
			priceChangePercent = (priceChange / prevClose) * 100
		}

		volumeMA := 0.0
		volumeRatio := 0.0

		// 计算成交量移动平均
		if i >= period-1 {
			sumVolume := 0.0
			for j := 0; j < period; j++ {
				vol := klines[i-j].Volume
				sumVolume += vol
			}
			volumeMA = sumVolume / float64(period)
			volumeRatio = volume / volumeMA

			volumeData = append(volumeData, &VolumeData{
				OpenTime:           kline.OpenTime,
				ClosePrice:         closePrice,
				Volume:             volume,
				VolumeMA:           volumeMA,
				VolumeRatio:        volumeRatio,
				PriceChange:        priceChange,
				PriceChangePercent: priceChangePercent,
			})
		}
	}
	return volumeData
}

// analyzeVolumeSignal 分析成交量数据，生成交易信号
func analyzeVolumeSignal(volumeData []*VolumeData, lookback int) Signal {
	if len(volumeData) < lookback+1 {
		return Signal{Target: "VOL", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	current := volumeData[len(volumeData)-1]
	// prev := volumeData[len(volumeData)-2]

	// 定义高成交量阈值 (例如，超过平均量的1.5倍)
	highVolumeThreshold := 1.5
	// 定义低成交量阈值 (例如，低于平均量的0.7倍)
	lowVolumeThreshold := 0.7

	// 信号1: 放量突破/跌破
	if current.VolumeRatio > highVolumeThreshold {
		if current.PriceChangePercent > 1.0 { // 价格显著上涨
			return Signal{
				Target:     "VOL",
				SignalType: "strong_bullish_breakout",
				Side:       "buy",
				Confidence: 0.8,
				Message:    "放量上涨！买盘强劲，强烈开多信号！",
			}
		} else if current.PriceChangePercent < -1.0 { // 价格显著下跌
			return Signal{
				Target:     "VOL",
				SignalType: "strong_bearish_breakout",
				Side:       "sell",
				Confidence: 0.8,
				Message:    "放量下跌！卖盘恐慌，强烈开空信号！",
			}
		}
	}

	// 信号2: 缩量回调/反弹
	if current.VolumeRatio < lowVolumeThreshold {
		// 在上升趋势中的缩量回调 (需要结合价格位置判断，这里简化)
		if current.PriceChangePercent < 0 {
			return Signal{
				Target:     "VOL",
				SignalType: "bullish_pullback",
				Side:       "buy",
				Confidence: 0.6,
				Message:    "缩量回调，卖压不足，潜在开多机会",
			}
		}
		// 在下跌趋势中的缩量反弹
		if current.PriceChangePercent > 0 {
			return Signal{
				Target:     "VOL",
				SignalType: "bearish_rally",
				Side:       "sell",
				Confidence: 0.6,
				Message:    "缩量反弹，买盘不济，潜在开空机会",
			}
		}
	}

	// 信号3: 量价背离检测 (简化版)
	if len(volumeData) >= 10 {
		// 检查顶背离：价格创新高，但成交量未创新高
		_, recentPriceHighIndex := findPriceHigh(volumeData, lookback)
		_, recentVolumeHighIndex := findVolumeHigh(volumeData, lookback)

		if recentPriceHighIndex == len(volumeData)-1 && recentVolumeHighIndex != len(volumeData)-1 {
			return Signal{
				Target:     "VOL",
				SignalType: "bearish_divergence",
				Side:       "sell",
				Confidence: 0.7,
				Message:    "量价顶背离：价格创新高但成交量未跟上，上涨动能减弱，警惕反转！",
			}
		}

		// 检查底背离：价格创新低，但成交量未创新低
		_, recentPriceLowIndex := findPriceLow(volumeData, lookback)
		_, recentVolumeLowIndex := findVolumeLow(volumeData, lookback)

		if recentPriceLowIndex == len(volumeData)-1 && recentVolumeLowIndex != len(volumeData)-1 {
			return Signal{
				Target:     "VOL",
				SignalType: "bullish_divergence",
				Side:       "buy",
				Confidence: 0.7,
				Message:    "量价底背离：价格创新低但成交量未放大，卖压衰竭，潜在底部！",
			}
		}
	}

	return Signal{Target: "VOL", SignalType: "none", Side: "none", Confidence: 0, Message: "未发现明确成交量信号"}
}

// 辅助函数：寻找价格高点
func findPriceHigh(data []*VolumeData, lookback int) (float64, int) {
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

// 辅助函数：寻找价格高点
func findPriceLow(data []*VolumeData, lookback int) (float64, int) {
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

// 辅助函数：寻找成交量高点
func findVolumeHigh(data []*VolumeData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	high := data[start].Volume
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].Volume > high {
			high = data[i].Volume
			index = i
		}
	}
	return high, index
}

// 辅助函数：寻找成交量高点
func findVolumeLow(data []*VolumeData, lookback int) (float64, int) {
	start := len(data) - lookback
	if start < 0 {
		start = 0
	}
	low := data[start].Volume
	index := start
	for i := start + 1; i < len(data); i++ {
		if data[i].Volume < low {
			low = data[i].Volume
			index = i
		}
	}
	return low, index
}

// 在主函数中调用
func GetVolumeMaSignal(klines []Kline, period int) Signal {
	// 1.	成交量先于价格：很多时候，成交量的异动会发生在价格大幅变动之前。密切关注“无量”和“天量”的异常情况。
	// 2.	结合价格位置和趋势：同样的放量，发生在价格高位和低位意义完全不同。高位放量多是派发（卖），低位放量多是吸筹（买）。
	// 3.	与关键价位结合：成交量分析在支撑位、阻力位、趋势线附近最为有效。在这些位置出现的放量突破/跌破，信号最可靠。
	// 4.	多时间框架验证：
	// o	在日线图上看到放量突破，确认大趋势。
	// o	在1小时图上寻找缩量回调的入场点。
	// 5.	“巨量”即“极端”：单一巨量K线往往意味着一个短期转折点，无论是“高潮顶点”还是“恐慌低点”。
	// 6.	BTC的特殊性：BTC市场24/7交易，且受新闻事件影响巨大。突然的放量很可能与特定新闻相关，需要判断这是否会改变市场的基本结构。

	// 计算成交量指标，使用20周期均线
	volumeData := calculateVolumeMA(klines, period)

	// 计算阻力位和支撑位
	// analysis, err := GetBTCAnalysis(klines)
	// if err != nil {
	// 	return Signal{Target: "VOL", SignalType: "none", Side: "none", Confidence: 0, Message: "数据计算失败"}
	// }

	// 分析成交量信号
	signal := analyzeVolumeSignal(volumeData, 10)

	return signal
}
