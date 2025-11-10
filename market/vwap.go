package market

// VWAPData 存储VWAP数据点
type VWAPData struct {
	OpenTime                     int64
	ClosePrice                   float64
	HighPrice                    float64
	LowPrice                     float64
	Volume                       float64
	CumulativeTypicalPriceVolume float64 // 累计（典型价格 * 成交量）
	CumulativeVolume             float64 // 累计成交量
	VWAP                         float64 // VWAP值
}

// calculateVWAP 计算VWAP指标
// 注意：此函数假设传入的K线数据是单个交易日内的，并且按时间排序
func calculateVWAP(klines []Kline, period int) []*VWAPData {
	var vwapList []*VWAPData
	var cumulativeTPV float64
	var cumulativeVol float64

	for i, kline := range klines {
		if i < period-1 {
			continue
		}
		// 解析K线数据
		high := kline.High
		low := kline.Low
		close := kline.Close
		volume := kline.Volume

		// 计算典型价格
		typicalPrice := (high + low + close) / 3.0
		// 计算典型价格 * 成交量
		typicalPriceVolume := typicalPrice * volume

		// 更新累计值
		cumulativeTPV += typicalPriceVolume
		cumulativeVol += volume

		// 计算VWAP
		var vwapValue float64
		if cumulativeVol > 0 {
			vwapValue = cumulativeTPV / cumulativeVol
		} else {
			vwapValue = 0
		}

		vwapList = append(vwapList, &VWAPData{
			OpenTime:                     kline.OpenTime,
			ClosePrice:                   close,
			HighPrice:                    high,
			LowPrice:                     low,
			Volume:                       volume,
			CumulativeTypicalPriceVolume: cumulativeTPV,
			CumulativeVolume:             cumulativeVol,
			VWAP:                         vwapValue,
		})
	}
	return vwapList
}

// 转换为字符串
func getVWAPDataString(vwapData []*VWAPData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(vwapData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := vwapData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.VWAP)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// analyzeVWAPSignal 分析VWAP数据，生成交易信号
func analyzeVWAPSignal(vwapData []*VWAPData, lookback int, period string) *Signal {
	if len(vwapData) < lookback+1 {
		return &Signal{Target: TargetVWAP, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	current := vwapData[len(vwapData)-1]
	prev := vwapData[len(vwapData)-2]

	// 信号1: 价格与VWAP的位置关系
	// 判断当前价格在VWAP之上还是之下
	priceAboveVWAP := current.ClosePrice > current.VWAP
	priceBelowVWAP := current.ClosePrice < current.VWAP

	// 判断之前价格在VWAP之上还是之下
	prevPriceAboveVWAP := prev.ClosePrice > prev.VWAP
	prevPriceBelowVWAP := prev.ClosePrice < prev.VWAP

	// 检测上穿：之前在下，现在在上
	if prevPriceBelowVWAP && priceAboveVWAP {
		return &Signal{
			Target:     TargetVWAP,
			SignalType: "bullish_crossover",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.7,
			Message:    "价格上穿VWAP，潜在开多信号",
		}
	}

	// 检测下穿：之前在上，现在在下
	if prevPriceAboveVWAP && priceBelowVWAP {
		return &Signal{
			Target:     TargetVWAP,
			SignalType: "bearish_crossover",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.7,
			Message:    "价格下穿VWAP，潜在开空信号",
		}
	}

	VWAPList := make([]float64, len(vwapData))
	priceList := make([]float64, len(vwapData))
	for i, vwap := range vwapData {
		VWAPList[i] = vwap.VWAP
		priceList[i] = vwap.ClosePrice
	}

	// 1. 判断A是否在B之上
	isAAboveB := CheckAAboveB(priceList, VWAPList)

	// 2. 判断VWAP是否在上扬
	slope, _ := LinearRegressionAnalysis(VWAPList)

	// 3. 计算相关性
	correlation := CalculateCorrelation(priceList, VWAPList)

	// 信号2: 趋势跟踪 - 价格在VWAP上方且VWAP本身在上扬
	if isAAboveB && slope > 0 && correlation > 0.5 {
		return &Signal{
			Target:     TargetVWAP,
			SignalType: "bullish_trend",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "价格在VWAP上方且VWAP上升，上升趋势健康",
		}
	}

	// 信号3: 趋势跟踪 - 价格在VWAP下方且VWAP本身在下降
	isAAboveB = CheckAAboveB(VWAPList, priceList)
	correlation = CalculateCorrelation(VWAPList, priceList)
	if isAAboveB && slope < 0 && correlation > 0.5 {
		return &Signal{
			Target:     TargetVWAP,
			SignalType: "bearish_trend",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "价格在VWAP下方且VWAP下降，下降趋势强劲",
		}
	}

	return &Signal{Target: TargetVWAP, SignalType: "none", Side: SideNone, Period: period, Confidence: 0.5, Message: "未发现明确信号，建议观望"}
}

// 在主函数中调用
func GetVWAPSignal(klines []Kline, period int, timePeriod string) *Signal {
	// 1.	VWAP是日内指标：VWAP在每个交易日结束时重置。因此，它对于日线图及以上的时间框架意义不大。它主要用于5分钟、15分钟、1小时等日内交易。
	// 2.	必须结合价格行为：VWAP提供的信号必须用K线形态来确认。在VWAP支撑处出现看涨Pin Bar，或在VWAP阻力处出现看跌吞没，会大大增加信号的可靠性。
	// 3.	结合成交量：当价格在VWAP处获得支撑或受到阻力时，如果伴随成交量的放大，则信号更强。

	// 4.	不要逆势操作：
	// o	当价格稳定在VWAP之上且VWAP线本身向上时，应主要寻找开多机会。
	// o	当价格被压制在VWAP之下且VWAP线本身向下时，应主要寻找开空机会。

	// 5.	设定合理的止损：
	// o	在VWAP支撑处开多，止损应设在VWAP线下方或近期低点下方。
	// o	在VWAP阻力处开空，止损应设在VWAP线上方或近期高点上方。

	// 6.	VWAP的局限性：在波动性极低的市场或横盘整理期间，VWAP可能会失去其作为动态支撑/阻力的意义。同时，它作为滞后指标，在趋势发生突变时，信号会来得较晚

	// 计算VWAP
	vwapData := calculateVWAP(klines, period)

	// 分析信号
	signal := analyzeVWAPSignal(vwapData, 5, timePeriod)

	return signal
}
