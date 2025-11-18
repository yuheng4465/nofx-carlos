package market

import (
	"sync"
	"time"
)

// FundingRateCache 资金费率缓存结构
// Binance Funding Rate 每 8 小时才更新一次，使用 1 小时缓存可显著减少 API 调用
type FundingRateCache struct {
	Rate      float64
	UpdatedAt time.Time
}

var (
	fundingRateMap sync.Map // map[string]*FundingRateCache
	frCacheTTL     = 1 * time.Hour
)

// 获取资金费费率信号
func getFundingRateSignal(fundingRate float64) *Signal {
	if fundingRate == 0 {
		return &Signal{
			Target:     TargetFundingRate,
			Side:       SideNone,
			Period:     Period3m,
			Confidence: 0.5,
			Message:    "资金费率0，中性",
		}
	}

	if fundingRate > 0 {
		return &Signal{
			Target:     TargetFundingRate,
			Side:       SideSell,
			Period:     Period3m,
			Confidence: 0.9,
			Message:    "资金费率大于0，开空信号",
		}
	} else {
		return &Signal{
			Target:     TargetFundingRate,
			Side:       SideBuy,
			Period:     Period3m,
			Confidence: 0.9,
			Message:    "资金费率小于0，开多信号",
		}
	}
}
