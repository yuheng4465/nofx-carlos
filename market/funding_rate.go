package market

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

// 获取策略所需数据
func getFundingRateCases(fundingRate float64) *Cases {
	casesData := &Cases{}
	casesData.Name = TargetMACD
	casesData.Metrics = map[string]interface{}{
		"fundingRate": fundingRate,
	}

	return casesData
}
