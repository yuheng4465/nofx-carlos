package market

// CMFData 存储CMF数据点
type CMFData struct {
	OpenTime        int64
	ClosePrice      float64
	HighPrice       float64
	LowPrice        float64
	Volume          float64
	MoneyFlowVolume float64 // 资金流体积
	CMF             float64 // CMF值
}

// calculateCMF 计算资金流量指标 (CMF)
// period: 计算周期，通常为20或21
func calculateCMF(klines []Kline, period int) []*CMFData {
	var cmfList []*CMFData
	var moneyFlowVolumes []float64
	var volumes []float64

	for i, kline := range klines {
		// 解析K线数据
		high := kline.High
		low := kline.Low
		close := kline.Close
		volume := kline.Volume

		// 计算资金流乘数
		moneyFlowMultiplier := ((close - low) - (high - close)) / (high - low)
		// 计算资金流体积
		moneyFlowVolume := moneyFlowMultiplier * volume

		moneyFlowVolumes = append(moneyFlowVolumes, moneyFlowVolume)
		volumes = append(volumes, volume)

		// 当积累足够的数据点时，开始计算CMF
		if i >= period-1 {
			sumMoneyFlowVolume := 0.0
			sumVolume := 0.0

			// 累计过去 'period' 期的数据
			for j := 0; j < period; j++ {
				sumMoneyFlowVolume += moneyFlowVolumes[i-j]
				sumVolume += volumes[i-j]
			}

			cmfValue := sumMoneyFlowVolume / sumVolume

			cmfList = append(cmfList, &CMFData{
				OpenTime:        kline.OpenTime,
				ClosePrice:      close,
				HighPrice:       high,
				LowPrice:        low,
				Volume:          volume,
				MoneyFlowVolume: moneyFlowVolume,
				CMF:             cmfValue,
			})
		}
	}
	return cmfList
}

// 转换为字符串
func getCMFDataString(cmfData []*CMFData, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(cmfData) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := cmfData[startIndex:]
	for _, v := range lastData {
		data = append(data, v.CMF)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// analyzeCMFSignal 分析CMF数据，生成交易信号
func analyzeCMFSignal(cmfData []*CMFData, period string) *Signal {
	if len(cmfData) < 5 {
		return &Signal{Target: TargetCMF, SignalType: "none", Side: SideNone, Period: period, Confidence: 0, Message: "数据不足"}
	}

	// 获取最近的数据点
	current := cmfData[len(cmfData)-1]
	prev := cmfData[len(cmfData)-2]

	// 信号1: 零轴穿越
	if prev.CMF <= 0 && current.CMF > 0 {
		return &Signal{
			Target:     TargetCMF,
			SignalType: "bullish_zero_cross",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.7,
			Message:    "CMF由下向上穿越零轴，资金开始净流入，潜在开多信号",
		}
	}
	if prev.CMF >= 0 && current.CMF < 0 {
		return &Signal{
			Target:     TargetCMF,
			SignalType: "bearish_zero_cross",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.7,
			Message:    "CMF由上向下穿越零轴，资金开始净流出，潜在开空信号",
		}
	}

	// 信号2: 强势区间判断
	if current.CMF > 0.05 { // 明显高于零轴
		return &Signal{
			Target:     TargetCMF,
			SignalType: "bullish_strong",
			Side:       SideBuy,
			Period:     period,
			Confidence: 0.6,
			Message:    "CMF持续在零轴上方，资金流入强劲，趋势看多",
		}
	}
	if current.CMF < -0.05 { // 明显低于零轴
		return &Signal{
			Target:     TargetCMF,
			SignalType: "bearish_strong",
			Side:       SideSell,
			Period:     period,
			Confidence: 0.6,
			Message:    "CMF持续在零轴下方，资金流出强劲，趋势看空",
		}
	}

	// 信号3: 超买超卖 (可作为反向信号，但需谨慎)
	if current.CMF > 0.25 {
		return &Signal{
			Target:     TargetCMF,
			SignalType: "overbought",
			Side:       SideWaring,
			Period:     period,
			Confidence: 0.4,
			Message:    "CMF显示超买，警惕回调，但强趋势中可能持续",
		}
	}
	if current.CMF < -0.25 {
		return &Signal{
			Target:     TargetCMF,
			SignalType: "oversold",
			Side:       SideWaring,
			Period:     period,
			Confidence: 0.4,
			Message:    "CMF显示超卖，警惕反弹，但强趋势中可能持续",
		}
	}

	return &Signal{Target: TargetCMF, SignalType: "none", Side: SideNone, Confidence: 0.5, Message: "未发现明确信号"}
}

// 根据1h和4h的CMF合并最终信号
func getCMFSignalWithMultiple(cmfSignal1h *Signal, cmfSIgnal4h *Signal) *Signal {
	// 4h定方向(如果方向背离则此项无效)
	Side4h := cmfSIgnal4h.Side
	Side1h := cmfSignal1h.Side
	Side := Side4h
	if Side1h != Side4h {
		return &Signal{
			Target:     TargetCMF,
			SignalType: "",
			Side:       SideWaring,
			Confidence: 0,
			Message:    "4h的CMF和1h方向相反，趋势矛盾",
		}
	}

	// 1h定强度
	confidence := cmfSignal1h.Confidence
	message := cmfSignal1h.Message
	return &Signal{
		Target:     TargetCMF,
		SignalType: "",
		Side:       Side,
		Confidence: confidence,
		Message:    message,
	}
}

func GetCmfSignal(klines []Kline, period int, tiemPeriod string) *Signal {
	// 1.	绝不单独使用CMF：CMF必须与价格行为分析、趋势线、支撑/阻力位以及其他指标（如均线、MACD）结合使用，进行多重验证。

	// 2.	时间框架选择：
	// o	日线/周线的CMF信号远比1小时/15分钟的信号可靠。
	// o	建议采用多时间框架分析：在大周期（日线）确定CMF的主要方向（在零上还是零下），在小周期（4小时/1小时）寻找具体的入场点。

	// 3.	背离信号的威力：CMF的顶背离和底背离是其最高质量的信号。但务必等待价格本身出现反转K线（如看跌吞没、黄昏之星）确认后再行动。
	// 4.	风险管理：无论CMF信号多么强，都必须设置止损。例如，在基于CMF上穿零轴开多时，将止损设置在近期震荡低点下方。
	// 5.	趋势是你的朋友：在CMF持续高于零轴的上升趋势中，应主要寻找开多机会；在CMF持续低于零轴的下降趋势中，应主要寻找开空机会。不要轻易逆势操作。

	// 计算CMF，通常使用20周期
	cmfData := calculateCMF(klines, period)

	// 分析信号
	signal := analyzeCMFSignal(cmfData, tiemPeriod)

	return signal
}
