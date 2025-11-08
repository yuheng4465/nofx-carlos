package market

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// OISignal 存储OI分析信号
type OISignal struct {
	SignalType string
	Confidence float64
	Message    string
}

// getOpenInterestData 获取OI数据（获取当前未平仓合约数）
func getOpenInterestData(symbol string) (*OIData, error) {
	api_url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 创建 HTTP 客户端并使用自定义 Transport
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(api_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Symbol:       result.Symbol,
		OpenInterest: oi,
		Timestamp:    result.Time,
	}, nil
}

// getOpenInterestHistData 获取OI趋势分析（合约持仓量历史）
func fetchOIData(symbol string, period string, limit int) ([]*OIData, error) {
	api_url := fmt.Sprintf("https://fapi.binance.com/futures/data/openInterestHist?symbol=%s&period=%s&limit=%d", symbol, period, limit)

	// 设置代理地址（例如：127.0.0.1:1080）
	proxyURL, err := url.Parse("http://127.0.0.1:8800")
	if err != nil {
		log.Fatalf("解析代理地址失败: %v", err)
	}

	// 创建自定义 Transport 并设置代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 创建 HTTP 客户端并使用自定义 Transport
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(api_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type InterestData struct {
		SumOpenInterest string `json:"sumOpenInterest"`
		Symbol          string `json:"symbol"`
		Timestamp       int64  `json:"timestamp"`
	}

	var data []InterestData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	history := make([]*OIData, len(data))
	for i, item := range data {
		history[i].OpenInterest, _ = strconv.ParseFloat(item.SumOpenInterest, 64)
		history[i].Timestamp = item.Timestamp
	}

	// 获取当前OI
	oIData, err := getOpenInterestData(symbol)
	if err != nil {
		return nil, err
	}

	// 当前数据合并到历史数据
	history = append(history, oIData)

	return history, nil
}

// analyzeOISignal 分析OI与价格关系，生成交易信号
func analyzeOISignal(oiData []*OIData, priceData []Kline, limit int) Signal {
	if limit < 2 || len(priceData) < 2 {
		return Signal{Target: "OI", SignalType: "none", Side: "none", Confidence: 0, Message: "数据不足"}
	}

	currentOI := oiData[limit-1]
	prevOI := oiData[limit-2]
	currentPrice := priceData[limit-1]
	prevPrice := priceData[limit-2]

	priceChangePercent := (currentPrice.Close - prevPrice.Close) / prevPrice.Close * 100
	oiChangePercent := (currentOI.OpenInterest - prevOI.OpenInterest) / prevOI.OpenInterest * 100

	// 信号1: 价涨量增 (最健康的多头信号)
	if priceChangePercent > 0.5 && oiChangePercent > 1.0 {
		return Signal{
			Target:     "OI",
			SignalType: "strong_bullish",
			Side:       "buy",
			Confidence: 0.9,
			Message:    "价涨量增！新开多单推动上涨，趋势健康，强烈开多信号",
		}
	}

	// 信号2: 价跌量增 (最健康的空头信号)
	if priceChangePercent < -0.5 && oiChangePercent > 1.0 {
		return Signal{
			Target:     "OI",
			SignalType: "strong_bearish",
			Side:       "sell",
			Confidence: 0.9,
			Message:    "价跌量增！新开空单推动下跌，趋势健康，强烈开空信号",
		}
	}

	// 信号3: 价涨量缩 (空头平仓推动的上涨，不可持续) 顶部背离,价格创出新高，但 O.I. 拒绝创新高甚至下降
	if priceChangePercent > 0.5 && oiChangePercent < -1.0 {
		return Signal{
			Target:     "OI",
			SignalType: "weak_bullish",
			Side:       "sell",
			Confidence: 0.6,
			Message:    "价涨量缩！上涨由空头平仓推动，动能不足，不宜追多",
		}
	}

	// 信号4: 价跌量缩 (多头平仓推动的下跌，可能见底)
	if priceChangePercent < -0.5 && oiChangePercent < -1.0 {
		return Signal{
			Target:     "OI",
			SignalType: "weak_bearish",
			Side:       "buy",
			Confidence: 0.6,
			Message:    "价跌量缩！下跌由多头平仓推动，卖压释放，警惕反弹",
		}
	}

	// 信号5: OI极端值预警
	if oiChangePercent > 10.0 {
		return Signal{
			Target:     "OI",
			SignalType: "oi_extreme",
			Side:       "none",
			Confidence: 0.7,
			Message:    "持仓量急剧变化！市场情绪极端，警惕大幅波动",
		}
	}

	return Signal{Target: "OI", SignalType: "none", Side: "none", Confidence: 0.5, Message: "未发现明确OI信号"}
}

// 在主函数中调用
func GetOISignal(symbol string, klines []Kline, limit int) Signal {
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

	// 使用15m数据
	oiData, err := fetchOIData(symbol, "15m", limit-1)
	if err != nil {
		return Signal{
			Target:     "OI",
			SignalType: "weak_bearish",
			Side:       "none",
			Confidence: 0,
			Message:    "获取OI数据失败",
		}
	}

	// 分析OI信号
	signal := analyzeOISignal(oiData, klines, limit)

	return signal
}
