package market

import "time"

// Data 市场数据结构
type Data struct {
	Symbol            string
	CurrentPrice      float64
	PriceChange1h     float64 // 1小时价格变化百分比
	PriceChange4h     float64 // 4小时价格变化百分比
	CurrentEMA20      float64
	CurrentMACD       float64
	CurrentRSI7       float64
	OpenInterest      []*OIData
	IntradaySeries    *IntradayData     // 3分钟数据 - 实时价格
	MidTermSeries15m  *MidTermData15m   // 15分钟数据 - 短期趋势
	MidTermSeries1h   *MidTermData1h    // 1小时数据 - 中期趋势
	LongerTermContext *LongerTermData   // 4小时数据 - 长期趋势
	longerTermData1d  *LongerTermData1d // 1天数据-长趋势
	SignalList        []*Signal         // 所有初步决策
	Signal            *Signal           // 最终决策
}

// OIData Open Interest数据
type OIData struct {
	Symbol       string  `json:"symbol"`
	Timestamp    int64   `json:"timestamp"`
	OpenInterest float64 `json:"openInterest"`
	Price        float64 `json:"price"`
}

// OBVData 存储成交量数据点
type OBVData struct {
	OpenTime   int64
	ClosePrice float64
	Volume     float64
	OBV        float64
}

// IntradayData 日内数据(3分钟间隔)
type IntradayData struct {
	MidPrices   []float64
	EMA20Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
	Volume      []float64
	ATR14       float64
	BSVOL       []*VolumeAnalysis // 主动买卖量
	EMA         []*EMAData        // EMA
	MACD        []*MACDData       // MACD
	RSI         []*RSIData        // RSI
	VWAP        []*VWAPData       // VWAP
	FundingRate float64
}

// MidTermData1h 日内信号(3m)
type IntradaySignalsData struct {
	BSVOL       *Signal // BSVOL
	EMA         *Signal // EMA
	MACD        *Signal // MACD
	RSI         *Signal // RSI
	VWAP        *Signal // VWAP
	FundingRate *Signal // 资金费率
}

// MidTermData15m 15分钟时间框架数据 - 短期趋势过滤
type MidTermData15m struct {
	MidPrices   []float64
	EMA20Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
	BOLL        []*BollingerBandData // BOLL
	OI          []*OIData            // OI
	EMA         []*EMAData           // EMA
	MACD        []*MACDData          // MACD
	RSI         []*RSIData           // RSI
	VOL         []*VolumeData        // VOL
	VWAP        []*VWAPData          // VWAP
}

// MidTermData1h 15分钟信号
type MidTermSignalsData15m struct {
	BOLL *Signal // BOLL
	EMA  *Signal // EMA
	OI   *Signal // OI
	MACD *Signal // MACD
	RSI  *Signal // RSI
	VOL  *Signal // VOL
	VWAP *Signal // VWAP
}

// MidTermData1h 1小时时间框架数据 - 中期趋势确认
type MidTermData1h struct {
	MidPrices   []float64
	EMA20Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
	BOLL        []*BollingerBandData // 布林带序列
	CMF         []*CMFData           // CMF
	EMA         []*EMAData           // EMA
	MACD        []*MACDData          // MACD
	OBV         []*OBVData           // OBV
	OI          []*OIData            // OI
	RSI         []*RSIData           // RSI
	TRIX        []*TRIXData          // TRIX
	VOL         []*VolumeData        // Vol
	VWAP        []*VWAPData          // VWAP
	WMA         []*WMAData           // WMA
}

// MidTermData1h 1小时信号
type MidTermSignalsData1h struct {
	BOLL *Signal // 布林带序列
	CMF  *Signal // CMF
	EMA  *Signal // EMA
	MACD *Signal // MACD
	OBV  *Signal // OBV
	OI   *Signal // OI
	RSI  *Signal // RSI
	TRIX *Signal // TRIX
	VOL  *Signal // Vol
	VWAP *Signal // VWAP
	WMA  *Signal // WMA

}

// LongerTermData 长期数据(4小时时间框架)
type LongerTermData struct {
	EMA20             float64 // EMA20
	EMA50             float64 // EMA50
	MidPrices         []float64
	ATR               []*ATRData           // ATR序列
	AVL               []*AVLData           // AVL序列
	BOLL              []*BollingerBandData // 布林带序列
	CCI               []*CCIData           // CCI
	CMF               []*CMFData           // CMF
	DMI               []*DMIData           // DMI
	EMA               []*EMAData           // EMA
	EMV               []*EMVData           // EMV
	KDJ               []*KDJData           // KDJ
	MACD              []*MACDData          // MACD
	MFI               []*MFIData           // MFI
	MTM               []*MTMData           // MTM
	OBV               []*OBVData           // OBV
	OI                []*OIData            // OI
	RSI               []*RSIData           // RSI
	SAR               []*SARData           // SAR
	StochRSI          []*StochRSIData      // StochRSI
	SupportResistance []*SupportResistance // SupportResistance
	TRIX              []*TRIXData          // TRIX
	VOL               []*VolumeData        // Vol
	WMA               []*WMAData           // WMA
	WR                []*WRData            // WR
}

// 长周期信号(4h)
type LongerTermSignalsData struct {
	ATR               *Signal // ATR
	AVL               *Signal // AVL
	BOLL              *Signal // 布林带
	CCI               *Signal // CCI
	CMF               *Signal // CMF
	DMI               *Signal // DMI
	EMA               *Signal // EMA
	EMV               *Signal // EMV
	KDJ               *Signal // KDJ
	MACD              *Signal // MACD
	MFI               *Signal // MFI
	MTM               *Signal // MTM
	OBV               *Signal // OBV
	OI                *Signal // OI
	RSI               *Signal // RSI
	SAR               *Signal // SAR
	StochRSI          *Signal // StochRSI
	SupportResistance *Signal // SupportResistance
	TRIX              *Signal // TRIX
	VOL               *Signal // Vol
	WMA               *Signal // WMA
	WR                *Signal // WR
}

// LongerTermData 长期数据(1天时间框架)
type LongerTermData1d struct {
	EMA20             float64 // EMA20
	EMA50             float64 // EMA50
	MidPrices         []float64
	ATR               []*ATRData           // ATR序列
	AVL               []*AVLData           // AVL序列
	BOLL              []*BollingerBandData // 布林带序列
	CCI               []*CCIData           // CCI
	CMF               []*CMFData           // CMF
	DMI               []*DMIData           // DMI
	EMA               []*EMAData           // EMA
	EMV               []*EMVData           // EMV
	KDJ               []*KDJData           // KDJ
	MACD              []*MACDData          // MACD
	MFI               []*MFIData           // MFI
	MTM               []*MTMData           // MTM
	OBV               []*OBVData           // OBV
	OI                []*OIData            // OI
	RSI               []*RSIData           // RSI
	SAR               []*SARData           // SAR
	StochRSI          []*StochRSIData      // StochRSI
	SupportResistance []*SupportResistance // SupportResistance
	TRIX              []*TRIXData          // TRIX
	VOL               []*VolumeData        // Vol
	WMA               []*WMAData           // WMA
	WR                []*WRData            // WR
}

// 长周期信号(1d)
type LongerTermSignalsData1d struct {
	ATR               *Signal // ATR
	AVL               *Signal // AVL
	BOLL              *Signal // 布林带
	CCI               *Signal // CCI
	CMF               *Signal // CMF
	DMI               *Signal // DMI
	EMA               *Signal // EMA
	EMV               *Signal // EMV
	KDJ               *Signal // KDJ
	MACD              *Signal // MACD
	MFI               *Signal // MFI
	MTM               *Signal // MTM
	OBV               *Signal // OBV
	OI                *Signal // OI
	RSI               *Signal // RSI
	SAR               *Signal // SAR
	StochRSI          *Signal // StochRSI
	SupportResistance *Signal // SupportResistance
	TRIX              *Signal // TRIX
	VOL               *Signal // Vol
	WMA               *Signal // WMA
	WR                *Signal // WR
}

// Binance API 响应结构
type ExchangeInfo struct {
	Symbols []SymbolInfo `json:"symbols"`
}

type SymbolInfo struct {
	Symbol            string `json:"symbol"`
	Status            string `json:"status"`
	BaseAsset         string `json:"baseAsset"`
	QuoteAsset        string `json:"quoteAsset"`
	ContractType      string `json:"contractType"`
	PricePrecision    int    `json:"pricePrecision"`
	QuantityPrecision int    `json:"quantityPrecision"`
}

type Kline struct {
	OpenTime            int64   `json:"openTime"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	CloseTime           int64   `json:"closeTime"`
	QuoteVolume         float64 `json:"quoteVolume"`
	Trades              int     `json:"trades"`
	TakerBuyBaseVolume  float64 `json:"takerBuyBaseVolume"`
	TakerBuyQuoteVolume float64 `json:"takerBuyQuoteVolume"`
}

type KlineResponse []interface{}

// 订单薄近期成交
type TradeInfo struct {
	Timestamp    int64  `json:"T"`
	Price        string `json:"p"`
	Quantity     string `json:"q"`
	IsBuyerMaker bool   `json:"m"` // 重要：true表示主动卖出，false表示主动买入
	TradeID      int64  `json:"a"`
}

type PriceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
}

// 特征数据结构
type SymbolFeatures struct {
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
	Price            float64   `json:"price"`
	PriceChange15Min float64   `json:"price_change_15min"`
	PriceChange1H    float64   `json:"price_change_1h"`
	PriceChange4H    float64   `json:"price_change_4h"`
	Volume           float64   `json:"volume"`
	VolumeRatio5     float64   `json:"volume_ratio_5"`
	VolumeRatio20    float64   `json:"volume_ratio_20"`
	VolumeTrend      float64   `json:"volume_trend"`
	RSI14            float64   `json:"rsi_14"`
	SMA5             float64   `json:"sma_5"`
	SMA10            float64   `json:"sma_10"`
	SMA20            float64   `json:"sma_20"`
	HighLowRatio     float64   `json:"high_low_ratio"`
	Volatility20     float64   `json:"volatility_20"`
	PositionInRange  float64   `json:"position_in_range"`
}

// 警报数据结构
type Alert struct {
	Type      string    `json:"type"`
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	AlertThresholds AlertThresholds `json:"alert_thresholds"`
	UpdateInterval  int             `json:"update_interval"` // seconds
	CleanupConfig   CleanupConfig   `json:"cleanup_config"`
}

type AlertThresholds struct {
	VolumeSpike      float64 `json:"volume_spike"`
	PriceChange15Min float64 `json:"price_change_15min"`
	VolumeTrend      float64 `json:"volume_trend"`
	RSIOverbought    float64 `json:"rsi_overbought"`
	RSIOversold      float64 `json:"rsi_oversold"`
}
type CleanupConfig struct {
	InactiveTimeout   time.Duration `json:"inactive_timeout"`    // 不活跃超时时间
	MinScoreThreshold float64       `json:"min_score_threshold"` // 最低评分阈值
	NoAlertTimeout    time.Duration `json:"no_alert_timeout"`    // 无警报超时时间
	CheckInterval     time.Duration `json:"check_interval"`      // 检查间隔
}

const (
	// 趋势方向
	SideBuy    = "buy"    // 开多
	SideSell   = "sell"   // 开空
	SideNone   = "none"   // 中性，无明确方向
	SideWaring = "waring" // 预警，可能反转

	// 数据周期
	Period3m  = "3m"  // 3分钟
	Period15m = "15m" // 15分钟
	Period1h  = "1h"  // 1小时
	Period4h  = "4h"  // 4小时
	Period1d  = "1d"  // 1天

	// 指标Target
	TargetATR               = "ATR"               // ATR
	TargetAVL               = "AVL"               // AVL
	TargetBOLL              = "BOLL"              // 布林带
	TargetBSVOL             = "BSVOL"             // BSVOL
	TargetCCI               = "CCI"               // CCI
	TargetCMF               = "CMF"               // CMF
	TargetDMI               = "DMI"               // DMI
	TargetEMA               = "EMA"               // EMA
	TargetEMV               = "EMV"               // EMV
	TargetFundingRate       = "FundingRate"       // 资金费率
	TargetKDJ               = "KDJ"               // KDJ
	TargetMACD              = "MACD"              // MACD
	TargetMFI               = "MFI"               // MFI
	TargetMTM               = "MTM"               // MTM
	TargetOBV               = "OBV"               // OBV
	TargetOI                = "OI"                // OI
	TargetRSI               = "RSI"               // RSI
	TargetSAR               = "SAR"               // SAR
	TargetStochRSI          = "StochRSI"          // StochRSI
	TargetSupportResistance = "SupportResistance" // SupportResistance
	TargetTRIX              = "TRIX"              // TRIX
	TargetVOL               = "VOL"               // Vol
	TargetVWAP              = "VWAP"              // VWAP
	TargetWMA               = "WMA"               // WMA
	TargetWR                = "WR"                // WR

)

// 分析信号
type Signal struct {
	Target     string    // 指标
	Period     string    // 数据时间周期
	SignalType string    // 信号类型
	Side       string    // 多空方向：buy开多，sell开空，none无方向，waring警报
	Confidence float64   // 信心
	Message    string    // 消息
	Strategy   *Strategy // 策略
	CaseData   *Cases    // 策略执行数据
}

// MarketState 定义市场状态
type MarketState string

const (
	TrendingBullish MarketState = "TrendingBullish" // 强劲上升趋势
	TrendingBearish MarketState = "TrendingBearish" // 强劲下跌趋势
	Ranging         MarketState = "Ranging"         // 震荡盘整
	TrendingWeak    MarketState = "TrendingWeak"    // 弱势趋势
)

// MarketAnalysis 存储市场分析结果
type MarketAnalysis struct {
	Timestamp      int64
	State          MarketState
	Confidence     float64
	ADX            float64
	PlusDI         float64
	MinusDI        float64
	EMASlope       float64
	BollingerWidth float64
	Message        string
}

// var config = Config{
// 	AlertThresholds: AlertThresholds{
// 		VolumeSpike:      3.0,
// 		PriceChange15Min: 0.05,
// 		VolumeTrend:      2.0,
// 		RSIOverbought:    70,
// 		RSIOversold:      30,
// 	},
// 	CleanupConfig: CleanupConfig{
// 		InactiveTimeout:   30 * time.Minute,
// 		MinScoreThreshold: 15.0,
// 		NoAlertTimeout:    20 * time.Minute,
// 		CheckInterval:     5 * time.Minute,
// 	},
// 	UpdateInterval: 60, // 1 minute
// }
