package market

// TradeDetail 存储逐笔成交数据
type TradeDetail struct {
	Symbol       string  `json:"symbol"`
	Timestamp    int64   `json:"timestamp"`
	Price        float64 `json:"price"`
	Quantity     float64 `json:"quantity"`
	IsBuyerMaker bool    `json:"isBuyerMaker"` // 重要：true表示主动卖出，false表示主动买入
	TradeID      int64   `json:"tradeId"`
}

// VolumeAnalysis 存储成交量分析结果
type VolumeAnalysis struct {
	Timestamp        int64   `json:"timestamp"`
	TotalVolume      float64 `json:"totalVolume"`
	ActiveBuyVolume  float64 `json:"activeBuyVolume"`  // 主动买入量
	ActiveSellVolume float64 `json:"activeSellVolume"` // 主动卖出量
	NetVolume        float64 `json:"netVolume"`        // 净主动量
	VolumeRatio      float64 `json:"volumeRatio"`      // 买卖量比率
	Price            float64 `json:"price"`
}

// analyzeTradeFlow 分析成交流水，计算主动买卖量
func analyzeTradeFlow(trades []TradeDetail, windowSize int) []*VolumeAnalysis {
	if len(trades) == 0 {
		return nil
	}

	var analysis []*VolumeAnalysis
	windowStart := 0

	for i := 1; i < len(trades); i++ {
		// 检查当前交易是否超出当前窗口的时间范围
		if trades[i].Timestamp-trades[windowStart].Timestamp >= int64(windowSize*1000) {
			// 分析从 windowStart 到 i-1 的窗口（不包含当前交易i）
			windowAnalysis := calculateWindowVolume(trades[windowStart:i])
			analysis = append(analysis, windowAnalysis)

			// 重要：更新窗口起始点为当前索引
			windowStart = i
		}
	}

	// 处理最后一个窗口（包含所有剩余交易）
	if windowStart < len(trades) {
		windowAnalysis := calculateWindowVolume(trades[windowStart:])
		analysis = append(analysis, windowAnalysis)
	}

	return analysis
}

// calculateWindowVolume 计算时间窗口内的成交量分析
func calculateWindowVolume(trades []TradeDetail) *VolumeAnalysis {
	var totalVolume, activeBuyVolume, activeSellVolume float64
	var latestPrice float64

	if len(trades) > 0 {
		latestPrice = trades[len(trades)-1].Price
	}

	for _, trade := range trades {
		totalVolume += trade.Quantity

		if trade.IsBuyerMaker {
			// 主动卖出
			activeSellVolume += trade.Quantity
		} else {
			// 主动买入
			activeBuyVolume += trade.Quantity
		}
	}

	netVolume := activeBuyVolume - activeSellVolume
	volumeRatio := 0.0
	if activeSellVolume > 0 {
		volumeRatio = activeBuyVolume / activeSellVolume
	} else if activeBuyVolume > 0 {
		volumeRatio = 10.0 // 极大值，表示只有主动买入
	}

	return &VolumeAnalysis{
		Timestamp:        trades[len(trades)-1].Timestamp,
		TotalVolume:      totalVolume,
		ActiveBuyVolume:  activeBuyVolume,
		ActiveSellVolume: activeSellVolume,
		NetVolume:        netVolume,
		VolumeRatio:      volumeRatio,
		Price:            latestPrice,
	}
}

// 转换为字符串
func getBSVOLDataString(volumeAnalysis []*VolumeAnalysis, period int) string {
	var data []float64
	// 取尾部数据
	startIndex := len(volumeAnalysis) - period
	if startIndex < 0 {
		startIndex = 0 // 如果数据不足10条，则从0开始取
	}
	lastData := volumeAnalysis[startIndex:]
	for _, v := range lastData {
		data = append(data, v.VolumeRatio)
	}
	// 将字节切片转换为字符串
	jsonString := formatFloatSlice(data)

	return jsonString
}

// 计算时间变化
// func getpriceChanges(klines []Kline, volumeAnalysis []VolumeAnalysis) {

// 	for _, volume := range volumeAnalysis {

// 	}
// }

// 获取策略所需数据
func getBSVOLCases(volumeAnalysis []*VolumeAnalysis) *StrategyData {
	if len(volumeAnalysis) < 2 {
		return &StrategyData{}
	}

	current := volumeAnalysis[len(volumeAnalysis)-1]
	prev := volumeAnalysis[len(volumeAnalysis)-2]

	casesData := &StrategyData{}
	casesData.Name = TargetMACD
	casesData.Metrics = map[string]interface{}{
		"currentPrice":            current.Price,
		"prevPrice":               prev.Price,
		"currentNetVolume":        current.NetVolume,
		"prevNetVolume":           prev.NetVolume,
		"currentVolumeRatio":      current.VolumeRatio,
		"currentTotalVolume":      current.TotalVolume,
		"prevTotalVolume":         prev.TotalVolume,
		"currentActiveSellVolume": current.ActiveSellVolume,
		"currentActiveBuyVolume":  current.ActiveBuyVolume,
		"prevActiveSellVolume":    prev.ActiveSellVolume,
		"prevActiveBuyVolume":     prev.ActiveBuyVolume,
	}

	return casesData
}

// generateBSVolumeSignal 生成主动买卖量交易信号
func generateBSVolumeSignal(volumeAnalysis []*VolumeAnalysis) *Signal {
	if len(volumeAnalysis) < 2 {
		return &Signal{Target: TargetBSVOL, SignalType: "none", Side: SideNone, Period: Period3m, Confidence: 0, Message: "数据不足"}
	}

	current := volumeAnalysis[len(volumeAnalysis)-1]
	prev := volumeAnalysis[len(volumeAnalysis)-2]

	// 计算价格变化
	priceChange := 0.0
	currentPrice := current.Price
	prevPrice := prev.Price
	priceChange = (currentPrice - prevPrice) / prevPrice * 100

	// 信号1: 量价齐升（健康上涨）
	if priceChange > 0.3 && current.NetVolume > 0 && current.VolumeRatio > 1.5 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "strong_bullish_breakout",
			Side:       SideBuy,
			Period:     Period3m,
			Confidence: 0.9,
			Message:    "价涨量增，主动买入主导！健康上涨趋势，开多信号",
		}
	}

	// 信号2: 量价齐跌（健康下跌）
	if priceChange < -0.3 && current.NetVolume < 0 && current.VolumeRatio < 0.7 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "strong_bearish_breakdown",
			Side:       SideSell,
			Period:     Period3m,
			Confidence: 0.9,
			Message:    "价跌量增，主动卖出主导！健康下跌趋势，开空信号",
		}
	}

	// 信号3: 放量滞涨（顶部信号）
	if priceChange < 0.1 && current.TotalVolume > prev.TotalVolume*1.5 && current.NetVolume < 0 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "distribution_top",
			Side:       SideSell,
			Period:     Period3m,
			Confidence: 0.8,
			Message:    "放量滞涨，主动卖出涌现！大资金派发，看空信号",
		}
	}

	// 信号4: 缩量止跌（底部信号）
	if priceChange > -0.1 && current.TotalVolume < prev.TotalVolume*0.7 && current.NetVolume > 0 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "accumulation_bottom",
			Side:       SideBuy,
			Period:     Period3m,
			Confidence: 0.8,
			Message:    "缩量止跌，主动买入承接！大资金吸筹，看多信号",
		}
	}

	// 信号5: 背离信号
	if len(volumeAnalysis) >= 10 {
		// 价格创新高但主动买入量未创新高
		if priceChange > 0 && current.NetVolume < prev.NetVolume {
			return &Signal{
				Target:     TargetBSVOL,
				SignalType: "bearish_divergence",
				Side:       SideSell,
				Period:     Period3m,
				Confidence: 0.7,
				Message:    "顶背离！价格创新高但主动买入力量减弱",
			}
		}

		// 价格创新低但主动卖出量未创新低
		if priceChange < 0 && current.NetVolume > prev.NetVolume {
			return &Signal{
				Target:     TargetBSVOL,
				SignalType: "bullish_divergence",
				Side:       SideBuy,
				Period:     Period3m,
				Confidence: 0.7,
				Message:    "底背离！价格创新低但主动卖出力量减弱",
			}
		}
	}

	// 信号6: 大单分析（简化版）
	if current.ActiveBuyVolume > current.ActiveSellVolume*3 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "large_buy_orders",
			Side:       SideBuy,
			Period:     Period3m,
			Confidence: 0.6,
			Message:    "大单买入明显，资金积极进场",
		}
	}

	if current.ActiveSellVolume > current.ActiveBuyVolume*3 {
		return &Signal{
			Target:     TargetBSVOL,
			SignalType: "large_sell_orders",
			Side:       SideSell,
			Period:     Period3m,
			Confidence: 0.6,
			Message:    "大单卖出明显，资金积极离场",
		}
	}

	return &Signal{Target: TargetBSVOL, SignalType: "none", Side: SideNone, Period: Period3m, Confidence: 0.5, Message: "未发现明确信号"}
}

func getBSVolSignal(simulatedTrades []TradeDetail, klines []Kline) *Signal {
	// 分析成交流水（30秒窗口）

	// 	1.	数据获取：
	// o	币安WebSocket Streams：<symbol>@trade 实时推送逐笔成交数据
	// o	REST API：/api/v3/trades 获取历史成交数据
	// o	关键字段：isBuyerMaker 判断主动方

	// 2.	主动买卖量的独特优势：
	// o	真实意图：区分主动攻击方和被动挂单方
	// o	微观洞察：看到订单流背后的真实故事
	// o	突破验证：确认突破是否由真实需求推动
	// o	吸收识别：发现大资金的隐藏意图

	// 3.	与其他指标配合：
	// o	订单簿：结合深度图看支撑阻力位的挂单情况
	// o	大单追踪：重点关注超过平均规模的大额成交
	// o	价格位置：在关键技术水平分析主动买卖量更有效

	// 4.	市场环境适应：
	// o	趋势市：跟随净主动量的方向
	// o	震荡市：在区间上下沿观察吸收现象
	// o	突破市：用主动买卖量确认突破真实性

	// 5.	风险管理要点：
	// o	假突破：突破时主动买卖量背离要警惕
	// o	流动性陷阱：注意大单可能只是为了触发止损
	// o	确认信号：等待价格行为确认后再入场

	// 6.	BTC市场的特殊应用：
	// o	BTC市场大单影响显著，要特别关注
	// o	机构操作通常在流动性好的时段进行
	// o	结合资金费率判断多空拥挤程度

	// 7.	避免的误区：
	// o	不要孤立看待单笔大额成交
	// o	不要忽略时间框架，不同周期意义不同
	// o	不要逆着主要趋势的净主动量方向交易

	volumeAnalysis := analyzeTradeFlow(simulatedTrades, 30)

	// 生成交易信号
	signal := generateBSVolumeSignal(volumeAnalysis)

	return signal
}
