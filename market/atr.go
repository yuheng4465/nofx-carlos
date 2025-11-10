package market

import (
	"math"
)

// ATRData 存储ATR数据点
type ATRData struct {
	OpenTime   int64
	HighPrice  float64
	LowPrice   float64
	ClosePrice float64
	TrueRange  float64
	ATR        float64
}

// calculateATR 计算平均真实波幅
func calculateATRData(klines []Kline, period int) []*ATRData {
	var atrList []*ATRData
	var trueRanges []float64

	for i, kline := range klines {
		// 计算真实波幅
		var trueRange float64
		if i == 0 {
			// 第一根K线，TR = High - Low
			trueRange = kline.High - kline.Low
		} else {
			prevClose := klines[i-1].Close
			method1 := kline.High - kline.Low           // 当日波幅
			method2 := math.Abs(kline.High - prevClose) // 跳空高开
			method3 := math.Abs(kline.Low - prevClose)  // 跳空低开
			trueRange = math.Max(method1, math.Max(method2, method3))
		}

		trueRanges = append(trueRanges, trueRange)

		// 计算ATR
		if i >= period-1 {
			sumTR := 0.0
			for j := 0; j < period; j++ {
				sumTR += trueRanges[i-j]
			}
			atrValue := sumTR / float64(period)

			atrList = append(atrList, &ATRData{
				OpenTime:   kline.OpenTime,
				HighPrice:  kline.High,
				LowPrice:   kline.Low,
				ClosePrice: kline.Close,
				TrueRange:  trueRange,
				ATR:        atrValue,
			})
		}
	}
	return atrList
}

// 转换为字符串
func getATRDataString(atrData []*ATRData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(atrData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := atrData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.ATR)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// analyzeATRTrend 分析ATR的趋势，辅助判断市场状态
func analyzeATRTrend(atrData []*ATRData, lookback int) *Signal {
	var signal Signal
	if len(atrData) < lookback+1 {
		return &Signal{Target: TargetATR, Period: Period4h, SignalType: SideNone, Confidence: 0, Message: "数据不足"}
	}

	currentATR := atrData[len(atrData)-1].ATR
	prevATR := atrData[len(atrData)-5].ATR // 看更早一点的ATR值

	threshold := 0.1 // 10%的变化阈值

	signal.Target = TargetATR
	signal.Period = Period4h
	signal.Side = SideNone
	if currentATR > prevATR*(1+threshold) {
		signal.SignalType = "volatility_expanding" // 波动性扩张
		signal.Message = "波动性正在扩张 - 趋势可能启动或加速"
		signal.Confidence = 0.7
	} else if currentATR < prevATR*(1-threshold) {
		signal.SignalType = "volatility_contracting" // 波动性收缩
		signal.Message = "波动性正在收缩 - 市场进入盘整，建议减少交易频率，或等待突破确认后再入场"
		signal.Confidence = 0.3
	} else {
		signal.SignalType = "volatility_stable" // 波动性稳定
		signal.Message = "波动性稳定，可按常规策略交易"
		signal.Confidence = 0.5
	}

	return &signal
}

func getATRSignal(klines []Kline, period int) *Signal {
	// 1、开多/开空的时机选择（过滤器）：
	// - 避免在ATR极低（波动收缩）时盲目交易，应等待波动性扩张伴随方向性突破。
	// - 利用ATR从低位抬头的时机，结合趋势指标捕捉趋势起始点。

	// 2、开多/开空的风险管理（核心工具）：
	// - 止损：使用 N * ATR 设置科学止损，避免被噪音震出局。
	// - 止盈：使用 M * ATR 设置合理目标，计算盈亏比。
	// - 仓位管理：ATR值大，说明波动大，应减小仓位；ATR值小，可适当增加仓位。

	// 	3.开多/开空的信心来源（趋势确认）：
	// o 一波强劲的趋势往往由ATR的显著扩张作为确认。
	// o ATR在趋势中下降是趋势动能减弱的警告，提示不要过度追涨杀跌。

	//ATR与趋势指标（如EMA, MACD, 布林带等）结合。让趋势指标决定方向，让ATR决定风险管理的参数和时机的过滤
	// 计算ATR，通常使用14周期（一般建议使用4h或者以上长周期计算）
	atrData := calculateATRData(klines, period)

	// 分析ATR趋势
	signal := analyzeATRTrend(atrData, 5)

	return signal
}
