package market

// OISignal 存储OI分析信号
type OISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// analyzeOISignal 分析OI与价格关系，生成交易信号
func analyzeOISignal(oiData []*OIData, priceData []Kline, peroid string) *Signal {
	if len(oiData) < 2 || len(priceData) < 2 {
		return &Signal{Target: TargetOI, SignalType: "none", Side: SideNone, Period: peroid, Confidence: 0, Message: "数据不足"}
	}

	currentOI := oiData[len(oiData)-1]
	prevOI := oiData[len(oiData)-2]
	currentPrice := priceData[len(oiData)-1]
	prevPrice := priceData[len(oiData)-2]

	priceChangePercent := (currentPrice.Close - prevPrice.Close) / prevPrice.Close * 100
	oiChangePercent := (currentOI.OpenInterest - prevOI.OpenInterest) / prevOI.OpenInterest * 100

	// 信号1: 价涨量增 (最健康的多头信号)
	if priceChangePercent > 0.5 && oiChangePercent > 1.0 {
		return &Signal{
			Target:     TargetOI,
			SignalType: "strong_bullish",
			Side:       SideBuy,
			Period:     peroid,
			Confidence: 0.9,
			Message:    "价涨量增！新开多单推动上涨，趋势健康，强烈开多信号",
		}
	}

	// 信号2: 价跌量增 (最健康的空头信号)
	if priceChangePercent < -0.5 && oiChangePercent > 1.0 {
		return &Signal{
			Target:     TargetOI,
			SignalType: "strong_bearish",
			Side:       SideSell,
			Period:     peroid,
			Confidence: 0.9,
			Message:    "价跌量增！新开空单推动下跌，趋势健康，强烈开空信号",
		}
	}

	// 信号3: 价涨量缩 (空头平仓推动的上涨，不可持续) 顶部背离,价格创出新高，但 O.I. 拒绝创新高甚至下降
	if priceChangePercent > 0.5 && oiChangePercent < -1.0 {
		return &Signal{
			Target:     TargetOI,
			SignalType: "weak_bullish",
			Side:       SideSell,
			Period:     peroid,
			Confidence: 0.6,
			Message:    "价涨量缩！上涨由空头平仓推动，动能不足，不宜追多",
		}
	}

	// 信号4: 价跌量缩 (多头平仓推动的下跌，可能见底)
	if priceChangePercent < -0.5 && oiChangePercent < -1.0 {
		return &Signal{
			Target:     TargetOI,
			SignalType: "weak_bearish",
			Side:       SideBuy,
			Period:     peroid,
			Confidence: 0.6,
			Message:    "价跌量缩！下跌由多头平仓推动，卖压释放，警惕反弹",
		}
	}

	// 信号5: OI极端值预警
	if oiChangePercent > 10.0 {
		return &Signal{
			Target:     TargetOI,
			SignalType: "oi_extreme",
			Side:       SideWaring,
			Period:     peroid,
			Confidence: 0.7,
			Message:    "持仓量急剧变化！市场情绪极端，警惕大幅波动",
		}
	}

	return &Signal{Target: TargetOI, SignalType: "none", Side: SideNone, Period: peroid, Confidence: 0.5, Message: "未发现明确OI信号"}
}

// 查询数据
func fetchOIData(symbol string, peroid string, limit int) ([]*OIData, error) {
	apiClient := NewAPIClient()
	oiHistoryData, err := apiClient.getOpenInterestHist(symbol, peroid, limit)
	if err != nil {
		return nil, err
	}
	oiData, err := apiClient.getOpenInterestData(symbol)
	if err != nil {
		return nil, err
	}

	oiHistoryData = append(oiHistoryData, oiData)

	return oiHistoryData, nil
}

// 转换为字符串
func getOIDataString(oiData []*OIData, period int) string {
	var data, newData []float64
	i := 0
	for _, v := range oiData {
		if v == nil {
			// fmt.Printf("Warning: oiData[%d] is nil, skipping.\n", i)
			continue
		}
		data = append(data, v.OpenInterest)
		i++
	}

	// 取尾部数据
	startIndex := len(data) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := data[startIndex:]
	for _, v := range lastData {
		newData = append(data, v)
	}

	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(newData)

	return jsonString
}

// 在主函数中调用
func getOISignal(symbol string, klines []Kline, peroid string, limit int) *Signal {
	// 1.	数据来源可靠性：
	// o	币安官方API对O.I.数据的支持可能有限，通常需要：
	// o	使用币安的WebSocket实时推送
	// o	访问币安官方网站的期货数据页面
	// o	使用专业的数据提供商（如Glassnode, CryptoQuant, Coinalyze）

	// 2.	O.I.的独特优势：
	// o	预见性：O.I.变化通常领先于价格趋势变化
	// o	真实性：反映真实的资金流向，难以操纵
	// o	情绪量化：直接量化市场多空双方的博弈强度

	// 3.	与其他指标配合：
	// o	资金费率：O.I.增加 + 高资金费率 = 多头拥挤，风险高
	// o	** liquidation数据**：O.I.极端值 + 大量爆仓 = 反转临近
	// o	成交量：确认O.I.变化的有效性

	// 4.	时间框架分析：
	// o	日线/周线O.I.：判断主要趋势和机构布局
	// o	4小时O.I.：寻找中期交易机会
	// o	1小时O.I.：捕捉短期波动和轧空/轧多机会

	// 5.	BTC市场的特殊应用：
	// o	季度合约交割：交割日前O.I.会自然下降，注意区分
	// o	重大事件：如ETF通过、减半等，关注O.I.的异常变化
	// o	杠杆周期：BTC市场特有的高杠杆导致的周期性清算

	// 6.	风险管理要点：
	// o	O.I.极端高值：市场过度拥挤，减少仓位
	// o	O.I.急剧下降：市场不确定性高，保持观望
	// o	结合资金费率：避免在极高资金费率时做多，极低时做空

	// 7.	避免的误区：
	// o	不要孤立使用O.I.，必须与价格结合分析
	// o	不要忽略不同交易所的O.I.差异
	// o	不要在所有时间框架使用同一套O.I.策略

	// 分析OI信号
	oiHistoryData, err := fetchOIData(symbol, peroid, limit)
	if err != nil {
		return &Signal{
			Target:     TargetOI,
			SignalType: "none",
			Period:     Period3m,
			Side:       SideNone,
			Confidence: 0,
			Message:    "数据不足",
		}
	}
	signal := analyzeOISignal(oiHistoryData, klines, peroid)

	return signal
}
