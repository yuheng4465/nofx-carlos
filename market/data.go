package market

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines3m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取3分钟K线数据 (最近10个)
	klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m") // 多获取一些用于计算
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// 获取15分钟K线数据 (最近40个) - 短期趋势
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据 (最近60个) - 中期趋势
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据 (最近60个) - 长期趋势
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h")
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 获取1d K线数据 (最近60个) - 长期趋势
	// klines1d, err = WSMonitorCli.GetCurrentKlines(symbol, "1d")
	// if err != nil {
	// 	return nil, fmt.Errorf("获取1天K线失败: %v", err)
	// }

	// 计算当前指标 (基于3分钟最新数据)
	currentPrice := klines3m[len(klines3m)-1].Close
	currentEMA20 := calculateEMA(klines3m, 20)
	currentMACD := calculateMACD(klines3m)
	currentRSI7 := calculateRSI(klines3m, 7)

	// 计算价格变化百分比
	// 1小时价格变化 = 20个3分钟K线前的价格
	priceChange1h := 0.0
	if len(klines3m) >= 21 { // 至少需要21根K线 (当前 + 20根前)
		price1hAgo := klines3m[len(klines3m)-21].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 计算日内系列数据 (3分钟)
	intradayData := calculateIntradaySeries(klines3m)

	// 计算长期数据
	// 计算15分钟系列数据
	midTermData15m := calculateMidTermSeries15m(klines15m, symbol)

	// 计算1小时系列数据
	midTermData1h := calculateMidTermSeries1h(klines1h, symbol)

	// 计算长期数据 (4小时)
	longerTermData := calculateLongerTermData(klines4h, symbol)

	// 计算长期数据（1天）
	// longerTermData1d := calculateLongerTermData1d(klines1d, symbol)

	longerTermSignalsData1d := &LongerTermSignalsData1d{}
	longerTermSignalsData := &LongerTermSignalsData{}
	midTermSignalsData1h := &MidTermSignalsData1h{}
	midTermSignalsData15m := &MidTermSignalsData15m{}
	intradaySignalsData := &IntradaySignalsData{}
	var signalList []*Signal

	// 1. 创建决策引擎
	engine := NewDecisionEngine()
	if err := engine.LoadConfig("default.json"); err != nil {
		return nil, fmt.Errorf("决策配置文件加载失败: %v", err)
	}

	// ATR
	// longerTermSignalsData.ATR = analyzeATRTrend(longerTermData.ATR, 5)
	// signalList = append(signalList, longerTermSignalsData.ATR)

	// AVL
	// longerTermSignalsData1d.AVL = analyzeAVLSignal(longerTermData1d.AVL, 10)
	// signalList = append(signalList, longerTermSignalsData1d.AVL)

	// BOLL
	// longerTermSignalsData1d.BOLL = analyzeBollingerSignal(longerTermData1d.BOLL, 10, Period1d)
	// midTermSignalsData1h.BOLL = analyzeBollingerSignal(midTermData1h.BOLL, 10, Period1h)
	longerTermSignalsData.BOLL = getStrategyResult(engine, TargetBOLL, getBollingerCases(longerTermData.BOLL, 10), Period4h)
	midTermSignalsData1h.BOLL = getStrategyResult(engine, TargetBOLL, getBollingerCases(midTermData1h.BOLL, 10), Period1h)
	midTermSignalsData15m.BOLL = getStrategyResult(engine, TargetBOLL, getBollingerCases(midTermData15m.BOLL, 10), Period15m)
	signalList = append(signalList, longerTermSignalsData.BOLL)
	signalList = append(signalList, midTermSignalsData1h.BOLL)
	signalList = append(signalList, midTermSignalsData15m.BOLL)

	// BS Vol
	// 获取最新成交数据
	var simulatedTrades []TradeDetail
	simulatedTrades, err = WSMonitorCli.GetCurrentTrades(symbol)
	if err != nil {
		return nil, fmt.Errorf("获取最新成交数据失败: %v", err)
	}
	volumeAnalysis := analyzeTradeFlow(simulatedTrades, 30)
	intradayData.BSVOL = volumeAnalysis
	// intradaySignalsData.BSVOL = generateBSVolumeSignal(volumeAnalysis, klines3m)
	intradaySignalsData.BSVOL = getStrategyResult(engine, TargetBSVOL, getBSVOLCases(volumeAnalysis, klines3m), Period3m)
	signalList = append(signalList, intradaySignalsData.BSVOL)

	// CCI
	// longerTermSignalsData.CCI = analyzeCCISignal(longerTermData.CCI, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.CCI)

	// CMF
	// longerTermSignalsData1d.CMF = analyzeCMFSignal(longerTermData1d.CMF, Period1d)
	// longerTermSignalsData.CMF = analyzeCMFSignal(longerTermData.CMF, Period4h)
	// signalList = append(signalList, longerTermSignalsData1d.CMF)
	// signalList = append(signalList, longerTermSignalsData.CMF)

	// DMI
	// longerTermSignalsData.DMI = analyzeDMISignal(longerTermData.DMI, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.DMI)

	// EMA
	// longerTermSignalsData1d.EMA = analyzeEMASignal(longerTermData1d.EMA, 5, Period1d)
	// longerTermSignalsData.EMA = analyzeEMASignal(longerTermData.EMA, 5, Period4h)
	// longerTermSignalsData1d.EMA = getStrategyResult(engine, TargetEMA, getEMACases(longerTermData1d.EMA, 5), Period1d)
	longerTermSignalsData.EMA = getStrategyResult(engine, TargetEMA, getEMACases(longerTermData.EMA, 5), Period4h)
	midTermSignalsData1h.EMA = getStrategyResult(engine, TargetEMA, getEMACases(midTermData1h.EMA, 5), Period1h)
	midTermSignalsData15m.EMA = getStrategyResult(engine, TargetEMA, getEMACases(midTermData15m.EMA, 5), Period15m)
	signalList = append(signalList, longerTermSignalsData1d.EMA)
	signalList = append(signalList, longerTermSignalsData.EMA)
	signalList = append(signalList, midTermSignalsData1h.EMA)
	signalList = append(signalList, midTermSignalsData15m.EMA)

	// EMV
	// longerTermSignalsData.EMV = analyzeEMVSignal(longerTermData.EMV, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.EMV)

	// KDJ
	// longerTermSignalsData.KDJ = analyzeKDJSignal(longerTermData.KDJ, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.KDJ)

	// MACD
	// longerTermSignalsData1d.MACD = analyzeMACDSignal(longerTermData1d.MACD, 20, Period1d)
	// longerTermSignalsData.MACD = analyzeMACDSignal(longerTermData.MACD, 20, Period4h)
	// midTermSignalsData1h.MACD = analyzeMACDSignal(midTermData1h.MACD, 20, Period1h)

	// longerTermSignalsData1d.MACD = getStrategyResult(engine, TargetMACD, getMACDCases(longerTermData1d.MACD, 20), Period1d)
	longerTermSignalsData.MACD = getStrategyResult(engine, TargetMACD, getMACDCases(longerTermData.MACD, 20), Period4h)
	midTermSignalsData1h.MACD = getStrategyResult(engine, TargetMACD, getMACDCases(midTermData1h.MACD, 20), Period1h)
	midTermSignalsData15m.MACD = getStrategyResult(engine, TargetMACD, getMACDCases(midTermData15m.MACD, 20), Period15m)
	signalList = append(signalList, longerTermSignalsData1d.MACD)
	signalList = append(signalList, longerTermSignalsData.MACD)
	signalList = append(signalList, midTermSignalsData1h.MACD)
	signalList = append(signalList, midTermSignalsData15m.MACD)

	// MFI
	// longerTermSignalsData.MFI = analyzeMFISignal(longerTermData.MFI, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.MFI)

	// MTM
	// longerTermSignalsData.MTM = analyzeMTMSignal(longerTermData.MTM, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.MTM)

	// OBV
	// longerTermSignalsData1d.OBV = analyzeOBVWithPeaks(longerTermData1d.OBV, 10, Period1d)
	// midTermSignalsData1h.OBV = analyzeOBVWithPeaks(midTermData1h.OBV, 10, Period1h)
	// signalList = append(signalList, longerTermSignalsData1d.OBV)
	// signalList = append(signalList, midTermSignalsData1h.OBV)

	// O.I.
	// longerTermSignalsData1d.OI = analyzeOISignal(longerTermData1d.OI, klines4h, Period1d)
	// longerTermSignalsData.OI = analyzeOISignal(longerTermData.OI, klines4h, Period4h)
	// midTermSignalsData1h.OI = analyzeOISignal(midTermData1h.OI, klines1h, Period1h)
	// longerTermSignalsData1d.OI = getStrategyResult(engine, TargetOI, getOICases(longerTermData1d.OI, klines1d), Period1d)
	longerTermSignalsData.OI = getStrategyResult(engine, TargetOI, getOICases(longerTermData.OI, klines4h), Period4h)
	midTermSignalsData1h.OI = getStrategyResult(engine, TargetOI, getOICases(midTermData1h.OI, klines1h), Period1h)
	midTermSignalsData15m.OI = getStrategyResult(engine, TargetOI, getOICases(midTermData15m.OI, klines15m), Period15m)
	signalList = append(signalList, longerTermSignalsData1d.OI)
	signalList = append(signalList, longerTermSignalsData.OI)
	signalList = append(signalList, midTermSignalsData1h.OI)
	signalList = append(signalList, midTermSignalsData15m.OI)

	//RSI
	// longerTermSignalsData1d.RSI = analyzeRSISignal(longerTermData1d.RSI, 20, Period1d)
	// longerTermSignalsData.RSI = analyzeRSISignal(longerTermData.RSI, 20, Period4h)
	// midTermSignalsData1h.RSI = analyzeRSISignal(midTermData1h.RSI, 20, Period1h)
	longerTermSignalsData.RSI = getStrategyResult(engine, TargetRSI, getRSICases(longerTermData.RSI, 20), Period4h)
	midTermSignalsData1h.RSI = getStrategyResult(engine, TargetRSI, getRSICases(midTermData1h.RSI, 20), Period1h)
	midTermSignalsData15m.RSI = getStrategyResult(engine, TargetRSI, getRSICases(midTermData15m.RSI, 20), Period15m)
	signalList = append(signalList, longerTermSignalsData.RSI)
	signalList = append(signalList, midTermSignalsData1h.RSI)
	signalList = append(signalList, midTermSignalsData15m.RSI)

	// SAR
	// longerTermSignalsData.SAR = analyzeSARSignal(longerTermData.SAR, 10, Period4h)
	// signalList = append(signalList, longerTermSignalsData.SAR)

	// STOCH RSI
	// longerTermSignalsData.StochRSI = analyzeStochRSISignal(longerTermData.StochRSI, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.StochRSI)

	//TRIX
	// longerTermSignalsData1d.TRIX = analyzeTRIXSignal(longerTermData1d.TRIX, 20, Period1d)
	// longerTermSignalsData.TRIX = analyzeTRIXSignal(longerTermData.TRIX, 20, Period4h)
	// midTermSignalsData1h.TRIX = analyzeTRIXSignal(midTermData1h.TRIX, 20, Period1h)
	// signalList = append(signalList, longerTermSignalsData1d.TRIX)
	// signalList = append(signalList, longerTermSignalsData.TRIX)
	// signalList = append(signalList, midTermSignalsData1h.TRIX)

	// VOL
	// longerTermSignalsData1d.VOL = analyzeVolumeSignal(longerTermData1d.VOL, 10, Period1d)
	// midTermSignalsData1h.VOL = analyzeVolumeSignal(midTermData1h.VOL, 10, Period1h)
	midTermSignalsData1h.VOL = getStrategyResult(engine, TargetVOL, getVOLCases(midTermData1h.VOL, 10), Period1h)
	midTermSignalsData15m.VOL = getStrategyResult(engine, TargetVOL, getVOLCases(midTermData15m.VOL, 10), Period15m)
	signalList = append(signalList, midTermSignalsData1h.VOL)
	signalList = append(signalList, midTermSignalsData15m.VOL)

	// VWAP
	// midTermSignalsData15m.VWAP = analyzeVWAPSignal(midTermData15m.VWAP, 5, Period15m)
	// signalList = append(signalList, midTermSignalsData15m.VWAP)

	// WMA
	// periods := []int{10, 30, 50} // 短期、中期、长期WMA
	// multiWMA1d := calculateMultiPeriodWMA(klines1d, periods)
	// wma10Data1d := multiWMA1d[10]
	// longerTermSignalsData1d.WMA = analyzeWMASignal(wma10Data1d, multiWMA1d, Period1d)
	// signalList = append(signalList, longerTermSignalsData1d.WMA)

	// multiWMA4h := calculateMultiPeriodWMA(klines4h, periods)
	// wma10Data4h := multiWMA4h[10]
	// longerTermSignalsData.WMA = analyzeWMASignal(wma10Data4h, multiWMA4h, Period4h)
	// signalList = append(signalList, longerTermSignalsData.WMA)

	// multiWMA1h := calculateMultiPeriodWMA(klines1h, periods)
	// wma10Data1h := multiWMA1h[10]
	// midTermSignalsData1h.WMA = analyzeWMASignal(wma10Data1h, multiWMA1h, Period1h)
	// signalList = append(signalList, midTermSignalsData1h.WMA)

	// WR
	// longerTermSignalsData.WR = analyzeWRSignal(longerTermData.WR, 20, Period4h)
	// signalList = append(signalList, longerTermSignalsData.WR)

	// 资金费率
	// 资金费率
	apiClient := NewAPIClient()
	fundingRate, _ := apiClient.getFundingRate(symbol)
	fundingRate = fundingRate * 100
	intradayData.FundingRate = fundingRate
	// intradaySignalsData.FundingRate = getFundingRateSignal(fundingRate)
	intradaySignalsData.FundingRate = getStrategyResult(engine, TargetFundingRate, getFundingRateCases(fundingRate), Period3m)
	signalList = append(signalList, intradaySignalsData.FundingRate)

	// 计算合并指标（针对多时间周期）
	signal := comprehensiveAnalysis(intradaySignalsData, midTermSignalsData15m, midTermSignalsData1h, longerTermSignalsData)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      midTermData1h.OI,
		IntradaySeries:    intradayData,
		MidTermSeries15m:  midTermData15m,
		MidTermSeries1h:   midTermData1h,
		LongerTermContext: longerTermData,
		// longerTermData1d:  longerTermData1d,
		Signal:     signal,
		SignalList: signalList,
	}, nil
}

// 获取决策执行结果
func getStrategyResult(engine *DecisionEngine, target string, data *Cases, period string) *Signal {
	matched, err := engine.ExecuteStrategy(target, data)
	if matched.Matched {
		return &Signal{
			Target:     target,
			SignalType: "none",
			Side:       matched.MatchedStrategy.Side,
			Period:     period,
			Confidence: matched.MatchedStrategy.Confidence,
			Strategy:   &matched.MatchedStrategy,
			CaseData:   data,
		}
	} else {
		return &Signal{
			Target:     target,
			SignalType: "none",
			Side:       SideNone,
			Period:     period,
			Confidence: 0,
			Message:    err.Error(),
		}
	}
}

// ComprehensiveAnalysis 综合决策引擎
func comprehensiveAnalysis(intradaySignalsData *IntradaySignalsData, midTermSignalsData15m *MidTermSignalsData15m,
	midTermSignalsData1h *MidTermSignalsData1h, longerTermSignalsData *LongerTermSignalsData) *Signal {

	// 步骤1: 大周期定方向
	trendDirection, trendConfidence := analyzeTrendDirection(longerTermSignalsData, intradaySignalsData, midTermSignalsData15m)

	// 步骤2: 中周期验动能
	momentumConfirmation, momentumMsg := analyzeMomentum(longerTermSignalsData, midTermSignalsData1h, trendDirection)

	// 步骤3: 小周期找点位
	entrySignal, entryMsg := analyzeEntrySignal(midTermSignalsData15m, trendDirection)

	// 综合决策
	var decision string
	var confidence float64
	var message string

	// 只有三者共振才交易
	if trendDirection != SideNone && momentumConfirmation && entrySignal != SideNone {
		if trendDirection == SideBuy && entrySignal == SideBuy {
			decision = SideBuy
			confidence = (trendConfidence + 0.7) / 2 // 综合计算置信度
			message = fmt.Sprintf("日线趋势看多 + 4H动能确认 + 1H出现做多信号。%s %s", momentumMsg, entryMsg)
		} else if trendDirection == SideSell && entrySignal == SideSell {
			decision = SideSell
			confidence = (trendConfidence + 0.7) / 2
			message = fmt.Sprintf("日线趋势看空 + 4H动能确认 + 1H出现做空信号。%s %s", momentumMsg, entryMsg)
		} else {
			decision = SideNone
			confidence = 0.5
			message = "趋势与入场信号矛盾，保持观望"
		}
	} else {
		decision = SideNone
		confidence = 0.5
		message = fmt.Sprintf("缺乏共振: 趋势=%s, 动能=%v, 入场=%s", trendDirection, momentumConfirmation, entrySignal)
	}

	return &Signal{
		Target:     "ALL",
		SignalType: "",
		Side:       decision,
		Confidence: confidence,
		Message:    message,
	}
}

// 步骤1: 分析大周期趋势
func analyzeTrendDirection(ds4 *LongerTermSignalsData, m3 *IntradaySignalsData, m15 *MidTermSignalsData15m) (string, float64) {
	bullishSignals := 0 // 看涨信号
	bearishSignals := 0 // 看跌信号

	bullishSignalsConfidence := 0.0 // 看涨信心分
	bearishSignalsConfidence := 0.0 // 看跌信心分

	// EMA排列
	// if d1.EMA20 > d1.EMA50 {
	// 	bullishSignals++
	// 	bullishSignalsConfidence += 0.8
	// } else {
	// 	bearishSignals++
	// 	bearishSignalsConfidence += 0.8
	// }

	// AVL(1d)
	// switch ds1.AVL.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds1.AVL.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds1.AVL.Confidence
	// }

	// 布林带位置(4h)
	switch ds4.BOLL.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.BOLL.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.BOLL.Confidence
	}

	// BS Vol(3m)
	switch m3.BSVOL.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += m3.BSVOL.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += m3.BSVOL.Confidence
	}

	// CCI(4h)
	// switch ds4.CCI.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.CCI.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.CCI.Confidence
	// }

	// CMF(1d)
	// switch ds1.CMF.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds1.CMF.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds1.CMF.Confidence
	// }

	// DMI趋势(4h)
	// switch ds4.DMI.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.DMI.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.DMI.Confidence
	// }

	// EMA(4h)
	switch ds4.EMA.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.EMA.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.EMA.Confidence
	}

	// EMV趋势(4h)
	// switch ds4.EMV.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.EMV.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.EMV.Confidence
	// }

	// KDJ趋势(4h)
	// switch ds4.KDJ.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.KDJ.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.KDJ.Confidence
	// }

	// MACD(4h)
	switch ds4.MACD.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.MACD.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.MACD.Confidence
	}

	// MFI(4h)
	// switch ds4.MFI.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.MFI.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.MFI.Confidence
	// }

	// MTM(4h)
	// switch ds4.MTM.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.MTM.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.MTM.Confidence
	// }

	// OBV(1d)
	// switch ds1.OBV.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds1.OBV.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds1.OBV.Confidence
	// }

	// OI(4h)
	switch ds4.OI.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.OI.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.OI.Confidence
	}

	// RSI(4h)
	switch ds4.RSI.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.RSI.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.RSI.Confidence
	}

	// SAR(4h)
	// switch ds4.SAR.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.SAR.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.SAR.Confidence
	// }

	// StochRSI(4h)
	// switch ds4.StochRSI.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.StochRSI.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.StochRSI.Confidence
	// }

	// TRIX(1d)
	// switch ds1.TRIX.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds1.TRIX.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds1.TRIX.Confidence
	// }

	// VOL(4h)
	switch ds4.VOL.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += ds4.VOL.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += ds4.VOL.Confidence
	}

	// VWAP(15m)
	// switch m15.VWAP.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += m15.VWAP.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += m15.VWAP.Confidence
	// }

	// WMA趋势(1d)
	// switch ds1.WMA.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds1.WMA.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds1.WMA.Confidence
	// }

	// WR(4h)
	// switch ds4.WR.Side {
	// case SideBuy:
	// 	bullishSignals++
	// 	bullishSignalsConfidence += ds4.WR.Confidence
	// case SideSell:
	// 	bearishSignals++
	// 	bearishSignalsConfidence += ds4.WR.Confidence
	// }

	// 资金费
	switch m3.FundingRate.Side {
	case SideBuy:
		bullishSignals++
		bullishSignalsConfidence += m3.FundingRate.Confidence
	case SideSell:
		bearishSignals++
		bearishSignalsConfidence += m3.FundingRate.Confidence
	}

	// 总计8项
	if bullishSignals >= 5 {
		return SideBuy, bullishSignalsConfidence / float64(bullishSignals)
	} else if bearishSignals >= 5 {
		return SideSell, bearishSignalsConfidence / float64(bearishSignals)
	}
	return SideNone, 0.5
}

// 步骤2: 分析中周期动能
func analyzeMomentum(h4 *LongerTermSignalsData, h1 *MidTermSignalsData1h, trend string) (bool, string) {
	if trend == SideNone {
		return false, "大周期无趋势"
	}

	mulSide := []string{h1.EMA.Side, h1.OI.Side, h1.MACD.Side, h1.RSI.Side}
	switch trend {
	case SideBuy:
		// 检查多头动能
		buyCount, _ := countBuySell(mulSide)
		if buyCount > 3 {
			return true, "中周期多头动能健康"
		}
	case SideSell:
		// 检查空头动能
		_, sellCount := countBuySell(mulSide)
		if sellCount > 3 {
			return true, "中周期空头动能健康"
		}
	}

	return false, "中周期动能与大周期趋势不匹配"
}

// 步骤3: 分析小周期入场信号
func analyzeEntrySignal(m15 *MidTermSignalsData15m, trend string) (string, string) {
	emaSide := m15.EMA.Side
	bollSide := m15.BOLL.Side // 价格回调至布林带中轨或以下
	macdSide := m15.MACD.Side
	rsiSide := m15.RSI.Side // 接近超卖但未极端
	switch trend {
	case SideBuy:
		// 做多条件：回调至支撑 + 指标转强
		if bollSide == SideBuy && macdSide == SideBuy && rsiSide == SideBuy && emaSide == SideBuy {
			return SideBuy, "价格回调至支撑位，多头动能重启"
		}
	case SideSell:
		// 做空条件：反弹至阻力 + 指标转弱
		if bollSide == SideSell && macdSide == SideSell && rsiSide == SideSell && emaSide == SideSell {
			return SideSell, "价格反弹至阻力位，空头动能重启"
		}
	}

	return SideNone, "未发现优质入场点"
}

func countBuySell(records []string) (buyCount int, sellCount int) {
	buyCount = 0
	sellCount = 0

	for _, record := range records {
		if record == SideBuy {
			buyCount++
		}
		if record == SideSell {
			sellCount++
		}
	}
	return buyCount, sellCount
}

// 分析当前市场状态
// 计算ADX
// 计算布林带
// 计算EMA
func analyzeMarketCondition(bbData []*BollingerBandData, dmiData []*DMIData, emaData []*EMAData) *MarketAnalysis {
	// EMA斜率
	emaSlope := emaData[len(emaData)-1].EMASlope

	//用ADX判断趋势强度
	currentDMI := dmiData[len(dmiData)-1]

	// 布林带带宽
	bbWidth := calculateBollingerWidth(bbData)

	// 决策逻辑
	var state MarketState
	var confidence float64
	var message string

	// 规则1: 用ADX判断趋势强度
	if currentDMI.ADX > 25 {
		// 趋势市场
		if currentDMI.PlusDI > currentDMI.MinusDI {
			state = TrendingBullish
			confidence = 0.8
			message = "ADX显示强劲上升趋势，建议顺势做多"
		} else {
			state = TrendingBearish
			confidence = 0.8
			message = "ADX显示强劲下降趋势，建议顺势做空"
		}

		// 用布林带宽度确认
		if bbWidth > 5.0 {
			confidence += 0.1
			message += "，布林带扩张确认趋势强度"
		}

	} else if currentDMI.ADX < 20 {
		// 震荡市场
		state = Ranging
		confidence = 0.7
		message = "ADX显示市场盘整，建议高抛低吸"

		// 用布林带宽度确认
		if bbWidth < 2.0 {
			confidence += 0.1
			message += "，布林带收缩确认震荡格局"
		}

	} else {
		// 弱势趋势或过渡期
		state = TrendingWeak
		confidence = 0.5
		message = "市场处于弱势趋势或方向选择期，建议谨慎操作"
	}

	// 规则2: 用EMA斜率过滤假信号
	if state == TrendingBullish && emaSlope < 0 {
		confidence -= 0.2
		message += "，但EMA斜率转弱需警惕"
	} else if state == TrendingBearish && emaSlope > 0 {
		confidence -= 0.2
		message += "，但EMA斜率转强需警惕"
	}

	// 规则3: 极端情况处理
	if bbWidth > 10.0 {
		message += "，波动率极高，注意风险管理"
	} else if bbWidth < 1.0 {
		message += "，波动率极低，警惕突破行情"
	}

	return &MarketAnalysis{
		State:          state,
		Confidence:     math.Min(confidence, 0.95), // 置信度上限
		ADX:            currentDMI.ADX,
		PlusDI:         currentDMI.PlusDI,
		MinusDI:        currentDMI.MinusDI,
		EMASlope:       emaSlope,
		BollingerWidth: bbWidth,
		Message:        message,
	}
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算3m ATR14
	data.ATR14 = calculateATR(klines, 14)

	return data
}

// calculateMidTermSeries15m 计算15分钟系列数据
func calculateMidTermSeries15m(klines []Kline, symbol string) *MidTermData15m {
	data := &MidTermData15m{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 15m指标
	data.OI, _ = fetchOIData(symbol, "15m", 20)
	data.EMA = calculateEMAData(klines, 20)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.RSI = calculateRSIData(klines, 14)
	data.VWAP = calculateVWAP(klines, 20)

	return data
}

// calculateMidTermSeries1h 计算1小时系列数据
func calculateMidTermSeries1h(klines []Kline, symbol string) *MidTermData1h {
	data := &MidTermData1h{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 1h指标
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.OBV = calculateOBV(klines)
	data.OI, _ = fetchOIData(symbol, "1h", 20)
	data.RSI = calculateRSIData(klines, 14)
	data.TRIX = calculateTRIX(klines, 15)
	data.VOL = calculateVolumeMA(klines, 20)
	// vwapData1h := calculateVWAP(klines1h, 20)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline, symbol string) *LongerTermData {
	data := &LongerTermData{}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	// data.ATR3 = calculateATR(klines, 3)
	// data.ATR7 = calculateATR(klines, 7)
	// data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	// if len(klines) > 0 {
	// 	data.CurrentVolume = klines[len(klines)-1].Volume
	// 	// 计算平均成交量
	// 	sum := 0.0
	// 	sum20 := 0.0
	// 	count := 0
	// 	for _, k := range klines {
	// 		sum += k.Volume
	// 		if count < 20 {
	// 			sum20 += k.Volume
	// 		}
	// 		count++
	// 	}
	// 	data.AverageVolume = sum / float64(len(klines))
	// 	// 最近20分成交均量
	// 	data.AverageVolume20 = sum20 / 20
	// }

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
	}

	// for i := start; i < len(klines); i++ {
	// 	if i >= 25 {
	// 		macd := calculateMACD(klines[:i+1])
	// 		data.MACDValues = append(data.MACDValues, macd)
	// 	}
	// 	if i >= 14 {
	// 		rsi14 := calculateRSI(klines[:i+1], 14)
	// 		data.RSI14Values = append(data.RSI14Values, rsi14)
	// 	}
	// }

	// 4h指标
	data.ATR = calculateATRData(klines, 14)
	// data.CCI = calculateCCI(klines, 20)
	// data.CMF = calculateCMF(klines, 20)
	// data.DMI = calculateDMI(klines, 14)
	data.EMA = calculateEMAData(klines, 20)
	// data.EMV = calculateEMV(klines, 14, 9)
	// data.KDJ = calculateKDJ(klines, 9, 3, 3)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	// data.MFI = calculateMFI(klines, 14)
	// data.MTM = calculateMTM(klines, 10, 6)
	data.OI, _ = fetchOIData(symbol, "4h", 20)
	data.RSI = calculateRSIData(klines, 14)
	// data.SAR = calculateSAR(klines, 0.02, 0.2, 0.02)
	// data.StochRSI = calculateStochRSI(klines, 14, 14, 3, 3)
	// data.TRIX = calculateTRIX(klines, 15)
	// data.WR = calculateWR(klines, 14)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData1d(klines []Kline, symbol string) *LongerTermData1d {
	data := &LongerTermData1d{}

	start := len(klines) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
	}

	// 1d指标
	data.AVL = calculateAVL(klines)
	data.BOLL = calculateBollingerBands(klines, 20, 2.0)
	data.CMF = calculateCMF(klines, 20)
	data.EMA = calculateEMAData(klines, 20)
	data.MACD = calculateMACDData(klines, 12, 26, 9)
	data.OBV = calculateOBV(klines)
	data.OI, _ = fetchOIData(symbol, "1d", 20)
	data.RSI = calculateRSIData(klines, 14)
	data.TRIX = calculateTRIX(klines, 15)
	data.VOL = calculateVolumeMA(klines, 20)

	return data
}

// getLastPrice 获取最新价格
func GetLastPrice(symbol string) (float64, error) {
	var klines3m []Kline
	klines3m, err := WSMonitorCli.GetCurrentKlines(symbol, "3m")
	if err != nil {
		return 0.0, fmt.Errorf("获取3分钟K线失败: %v", err)
	}
	CurrentPrice := klines3m[len(klines3m)-1].Close

	return CurrentPrice, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	sb.WriteString(fmt.Sprintf("当前: Price = %s, EMA20 = %.3f, MACD = %.3f, RSI7 = %.3f\n\n",
		priceStr, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	if len(data.SignalList) > 0 {
		sb.WriteString("CheckList:\n")
		sb.WriteString("| Index| Period | Side | Confidence | Message |\n")
		sb.WriteString("| -----| ----- | ----- | ----- | ----- |\n")
		for _, s := range data.SignalList {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.1f | %s |\n", s.Target, s.Period, s.Side, s.Confidence, s.Message))
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.1f | %s |\n", data.Signal.Target, data.Signal.Period, data.Signal.Side, data.Signal.Confidence, data.Signal.Message))
		sb.WriteString("| -----|  ----- |----- | ----- | ----- |\n\n")

		sb.WriteString("StrategyList:\n")
		sb.WriteString("| Target | Name | Side | Confidence | Expression | Message |\n\n")
		for _, s := range data.SignalList {
			if s.Strategy.Name != "" {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.1f | %s | %s |\n", s.Target, s.Strategy.Name, s.Strategy.Side, s.Strategy.Confidence, s.Strategy.Expression, s.Message))
			}
			sb.WriteString("| -----|  ----- |----- | ----- | ----- |\n\n")
		}

		sb.WriteString("StrategyCasesData:\n")
		for _, s := range data.SignalList {
			sb.WriteString(fmt.Sprintf("%s:\n %s\n", s.Target, formatStringStruct(s.CaseData.Metrics)))
		}
	}

	sb.WriteString("指标数据: \n\n")
	sb.WriteString("| Index| Period | Data |\n")
	sb.WriteString("| -----| ----- | ----- |\n")
	// 1d指标
	// sb.WriteString(fmt.Sprintf("| AVL | 1d | %s |\n", getAVLDataString(data.longerTermData1d.AVL, 10)))
	// sb.WriteString(fmt.Sprintf("| BOLL | 1d | %s |\n", getBOLLDataString(data.longerTermData1d.BOLL, 10)))
	// sb.WriteString(fmt.Sprintf("| CMF | 1d | %s |\n", getCMFDataString(data.longerTermData1d.CMF, 10)))
	// sb.WriteString(fmt.Sprintf("| EMA20 | 1d | %s |\n", getEMADataString(data.longerTermData1d.EMA, 10)))
	// sb.WriteString(fmt.Sprintf("| MACD | 1d | %s |\n", getMACDDataString(data.longerTermData1d.MACD, 10)))
	// sb.WriteString(fmt.Sprintf("| OBV | 1d | %s |\n", getOBVDataString(data.longerTermData1d.OBV, 10)))
	// sb.WriteString(fmt.Sprintf("| OI | 1d | %s |\n", getOIDataString(data.longerTermData1d.OI, 10)))
	// sb.WriteString(fmt.Sprintf("| RSI14 | 1d | %s |\n", getRSIDataString(data.longerTermData1d.RSI, 10)))
	// sb.WriteString(fmt.Sprintf("| TRIX | 1d | %s |\n", getTRIXDataString(data.longerTermData1d.TRIX, 10)))
	// sb.WriteString(fmt.Sprintf("| VOL | 1d | %s |\n", getVolDataString(data.longerTermData1d.VOL, 10)))
	// sb.WriteString(fmt.Sprintf("| Price | 1d | %s |\n", formatFloatSlice(data.longerTermData1d.MidPrices)))

	// 4h指标
	sb.WriteString(fmt.Sprintf("| ATR | 4h | %s |\n", getATRDataString(data.LongerTermContext.ATR, 10)))
	// sb.WriteString(fmt.Sprintf("| CCI | 4h | %s |\n", getCCIDataString(data.LongerTermContext.CCI, 10)))
	sb.WriteString(fmt.Sprintf("| CMF | 4h | %s |\n", getCMFDataString(data.LongerTermContext.CMF, 10)))
	// sb.WriteString(fmt.Sprintf("| DMI | 4h | %s |\n", getDMIDataString(data.LongerTermContext.DMI, 10)))
	sb.WriteString(fmt.Sprintf("| EMA20 | 4h | %s |\n", getEMADataString(data.LongerTermContext.EMA, 10)))
	// sb.WriteString(fmt.Sprintf("| EMV | 4h | %s |\n", getEMVDataString(data.LongerTermContext.EMV, 10)))
	// sb.WriteString(fmt.Sprintf("| KDJ | 4h | %s |\n", getKDJDataString(data.LongerTermContext.KDJ, 10)))
	sb.WriteString(fmt.Sprintf("| MACD | 4h | %s |\n", getMACDDataString(data.LongerTermContext.MACD, 10)))
	// sb.WriteString(fmt.Sprintf("| MFI | 4h | %s |\n", getMFIDataString(data.LongerTermContext.MFI, 10)))
	sb.WriteString(fmt.Sprintf("| OI | 4h | %s |\n", getOIDataString(data.LongerTermContext.OI, 10)))
	sb.WriteString(fmt.Sprintf("| RSI14 | 4h | %s |\n", getRSIDataString(data.LongerTermContext.RSI, 10)))
	// sb.WriteString(fmt.Sprintf("| SAR | 4h | %s |\n", getSARDataString(data.LongerTermContext.SAR, 10)))
	// sb.WriteString(fmt.Sprintf("| StochRSI | 4h | %s |\n", getStochRSIDataString(data.LongerTermContext.StochRSI, 10)))
	// sb.WriteString(fmt.Sprintf("| TRIX | 4h | %s |\n", getTRIXDataString(data.LongerTermContext.TRIX, 10)))
	// sb.WriteString(fmt.Sprintf("| WR | 4h | %s |\n", getWRDataString(data.LongerTermContext.WR, 10)))
	sb.WriteString(fmt.Sprintf("| Price | 4h | %s |\n", formatFloatSlice(data.LongerTermContext.MidPrices)))

	// 1h指标
	sb.WriteString(fmt.Sprintf("| BOLL | 1h | %s |\n", getBOLLDataString(data.MidTermSeries1h.BOLL, 10)))
	sb.WriteString(fmt.Sprintf("| MACD | 1h | %s |\n", getMACDDataString(data.MidTermSeries1h.MACD, 10)))
	sb.WriteString(fmt.Sprintf("| OBV | 1h | %s |\n", getOBVDataString(data.MidTermSeries1h.OBV, 10)))
	sb.WriteString(fmt.Sprintf("| OI | 1h | %s |\n", getOIDataString(data.MidTermSeries1h.OI, 10)))
	// sb.WriteString(fmt.Sprintf("| TRIX | 1h | %s |\n", getTRIXDataString(data.MidTermSeries1h.TRIX, 10)))
	sb.WriteString(fmt.Sprintf("| VOL | 1h | %s |\n", getVolDataString(data.MidTermSeries1h.VOL, 10)))
	sb.WriteString(fmt.Sprintf("| Price | 1h | %s |\n", formatFloatSlice(data.MidTermSeries1h.MidPrices)))
	sb.WriteString(fmt.Sprintf("| EMA20 | 1h | %s |\n\n", formatFloatSlice(data.MidTermSeries1h.EMA20Values)))
	sb.WriteString(fmt.Sprintf("| RSI7 | 1h | %s |\n\n", formatFloatSlice(data.MidTermSeries1h.RSI7Values)))
	sb.WriteString(fmt.Sprintf("| RSI14 | 1h | %s |\n\n", formatFloatSlice(data.MidTermSeries1h.RSI14Values)))

	// 15m指标
	sb.WriteString(fmt.Sprintf("| BSVOL | 15m | %s |\n", getBSVOLDataString(data.IntradaySeries.BSVOL, 10)))
	// sb.WriteString(fmt.Sprintf("| VWAP | 15m | %s |\n", getVWAPDataString(data.MidTermSeries15m.VWAP, 10)))
	sb.WriteString(fmt.Sprintf("| Price | 15m | %s |\n", formatFloatSlice(data.MidTermSeries15m.MidPrices)))
	sb.WriteString(fmt.Sprintf("| EMA20 | 15m | %s |\n\n", formatFloatSlice(data.MidTermSeries15m.EMA20Values)))
	sb.WriteString(fmt.Sprintf("| MACD | 15m | %s |\n\n", formatFloatSlice(data.MidTermSeries15m.MACDValues)))
	sb.WriteString(fmt.Sprintf("| RSI7 | 15m | %s |\n\n", formatFloatSlice(data.MidTermSeries15m.RSI7Values)))
	sb.WriteString(fmt.Sprintf("| RSI14 | 15m | %s |\n\n", formatFloatSlice(data.MidTermSeries15m.RSI14Values)))

	// 3m指标
	sb.WriteString(fmt.Sprintf("| FundingRate | 3m | %.8f %%|\n", data.IntradaySeries.FundingRate))
	sb.WriteString(fmt.Sprintf("| Price | 3m | %s |\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
	sb.WriteString(fmt.Sprintf("| EMA20 | 3m | %s |\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
	sb.WriteString(fmt.Sprintf("| MACD | 3m | %s |\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
	sb.WriteString(fmt.Sprintf("| RSI7 | 3m | %s |\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
	sb.WriteString(fmt.Sprintf("| RSI14 | 3m | %s |\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
	sb.WriteString("| -----| ----- | ----- |\n\n")

	// if data.IntradaySeries != nil {
	// 	sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

	// 	if len(data.IntradaySeries.MidPrices) > 0 {
	// 		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
	// 	}

	// 	if len(data.IntradaySeries.EMA20Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
	// 	}

	// 	if len(data.IntradaySeries.MACDValues) > 0 {
	// 		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
	// 	}

	// 	if len(data.IntradaySeries.RSI7Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
	// 	}

	// 	if len(data.IntradaySeries.RSI14Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
	// 	}
	// }

	// if data.MidTermSeries15m != nil {
	// 	sb.WriteString("Mid‑term series (15‑minute intervals, oldest → latest):\n\n")

	// 	if len(data.MidTermSeries15m.MidPrices) > 0 {
	// 		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MidPrices)))
	// 	}

	// 	if len(data.MidTermSeries15m.EMA20Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.EMA20Values)))
	// 	}

	// 	if len(data.MidTermSeries15m.MACDValues) > 0 {
	// 		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries15m.MACDValues)))
	// 	}

	// 	if len(data.MidTermSeries15m.RSI7Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI7Values)))
	// 	}

	// 	if len(data.MidTermSeries15m.RSI14Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries15m.RSI14Values)))
	// 	}
	// }

	// if data.MidTermSeries1h != nil {
	// 	sb.WriteString("Mid‑term series (1‑hour intervals, oldest → latest):\n\n")

	// 	if len(data.MidTermSeries1h.MidPrices) > 0 {
	// 		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MidPrices)))
	// 	}

	// 	if len(data.MidTermSeries1h.EMA20Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.EMA20Values)))
	// 	}

	// 	if len(data.MidTermSeries1h.MACDValues) > 0 {
	// 		sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.MidTermSeries1h.MACDValues)))
	// 	}

	// 	if len(data.MidTermSeries1h.RSI7Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI7Values)))
	// 	}

	// 	if len(data.MidTermSeries1h.RSI14Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.MidTermSeries1h.RSI14Values)))
	// 	}
	// }

	// if data.LongerTermContext != nil {
	// 	sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

	// 	sb.WriteString(fmt.Sprintf("EMA20: %.3f vs. 50‑Period EMA: %.3f\n\n",
	// 		data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

	// 	if len(data.LongerTermContext.MACDValues) > 0 {
	// 		sb.WriteString(fmt.Sprintf("MACD: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
	// 	}

	// 	if len(data.LongerTermContext.RSI14Values) > 0 {
	// 		sb.WriteString(fmt.Sprintf("RSI14): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
	// 	}

	// 	if len(data.LongerTermContext.ATR) > 0 {
	// 		sb.WriteString(fmt.Sprintf("ATR14: %.4f\n\n", data.LongerTermContext.ATR[len(data.LongerTermContext.ATR)-1].ATR))
	// 	}
	// }

	return sb.String()
}

func formatStringStruct[T any](slice T) string {
	jsonData, err := json.Marshal(slice)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return ""
	}

	// 将字节切片转换为字符串
	jsonString := string(jsonData)

	return jsonString
}

// checkAAboveB 检查A是否在B之上
func CheckAAboveB(dataA, dataB []float64) bool {
	aboveCount := 0
	totalPoints := len(dataA)

	for i := range totalPoints {
		if dataA[i] > dataB[i] {
			aboveCount++
		}
	}

	// 如果A在B之上的点数超过80%，认为A在B之上
	return float64(aboveCount)/float64(totalPoints) > 0.8
}

// LinearRegressionAnalysis 线性回归分析
func LinearRegressionAnalysis(data []float64) (slope, strength float64) {
	n := float64(len(data))
	var sumX, sumY, sumXY, sumXX float64

	for i, price := range data {
		x := float64(i)
		y := price
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// 计算斜率
	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 计算趋势强度 (R-squared)
	meanY := sumY / n
	var totalSS, regSS float64
	for i, price := range data {
		x := float64(i)
		predicted := (slope * x) + (sumY/n - slope*sumX/n)
		totalSS += math.Pow(price-meanY, 2)
		regSS += math.Pow(predicted-meanY, 2)
	}

	rSquared := 0.0
	if totalSS > 0 {
		rSquared = regSS / totalSS
	}

	// 斜率大于0表示上扬，小于0表示下跌，趋势强度使用R-squared
	return slope, rSquared
}

// calculateAvgDistance 计算A和B之间的平均距离
func CalculateAvgDistance(dataA, dataB []float64) float64 {
	var totalDistance float64
	for i := 0; i < len(dataA); i++ {
		totalDistance += dataA[i] - dataB[i]
	}
	return totalDistance / float64(len(dataA))
}

// calculateCorrelation 计算A和B的相关性
func CalculateCorrelation(dataA, dataB []float64) float64 {
	n := float64(len(dataA))

	var sumA, sumB, sumAB, sumA2, sumB2 float64
	for i := range dataA {
		a := dataA[i]
		b := dataB[i]
		sumA += a
		sumB += b
		sumAB += a * b
		sumA2 += a * a
		sumB2 += b * b
	}

	numerator := n*sumAB - sumA*sumB
	denominator := math.Sqrt((n*sumA2 - sumA*sumA) * (n*sumB2 - sumB*sumB))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// isStaleData detects stale data (consecutive price freeze)
// Fix DOGEUSDT-style issue: consecutive N periods with completely unchanged prices indicate data source anomaly
func isStaleData(klines []Kline, symbol string) bool {
	if len(klines) < 5 {
		return false // Insufficient data to determine
	}

	// Detection threshold: 5 consecutive 3-minute periods with unchanged price (15 minutes without fluctuation)
	const stalePriceThreshold = 5
	const priceTolerancePct = 0.0001 // 0.01% fluctuation tolerance (avoid false positives)

	// Take the last stalePriceThreshold K-lines
	recentKlines := klines[len(klines)-stalePriceThreshold:]
	firstPrice := recentKlines[0].Close

	// Check if all prices are within tolerance
	for i := 1; i < len(recentKlines); i++ {
		priceDiff := math.Abs(recentKlines[i].Close-firstPrice) / firstPrice
		if priceDiff > priceTolerancePct {
			return false // Price fluctuation exists, data is normal
		}
	}

	// Additional check: MACD and volume
	// If price is unchanged but MACD/volume shows normal fluctuation, it might be a real market situation (extremely low volatility)
	// Check if volume is also 0 (data completely frozen)
	allVolumeZero := true
	for _, k := range recentKlines {
		if k.Volume > 0 {
			allVolumeZero = false
			break
		}
	}

	if allVolumeZero {
		log.Printf("⚠️  %s stale data confirmed: price freeze + zero volume", symbol)
		return true
	}

	// Price frozen but has volume: might be extremely low volatility market, allow but log warning
	log.Printf("⚠️  %s detected extreme price stability (no fluctuation for %d consecutive periods), but volume is normal", symbol, stalePriceThreshold)
	return false
}
