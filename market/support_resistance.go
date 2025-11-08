package market

import (
	"fmt"
	"sort"
	"time"
)

// 价格水平
type PriceLevel struct {
	Price    float64 `json:"price"`
	Strength float64 `json:"strength"` // 强度 0-1
	Type     string  `json:"type"`     // resistance/support
	Volume   float64 `json:"volume"`   // 成交量
	Touches  int     `json:"touches"`  // 触碰次数
}

// 支撑阻力分析结果
type SupportResistance struct {
	Supports     []PriceLevel `json:"supports"`    // 支撑位
	Resistances  []PriceLevel `json:"resistances"` // 阻力位
	CurrentPrice float64      `json:"current_price"`
	Timestamp    int64        `json:"timestamp"`
}

// 转换K线数据为价格数组
func ConvertKlinesToPrices(klines []Kline) ([]float64, []float64, []float64, []float64) {
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	closes := make([]float64, len(klines))
	volumes := make([]float64, len(klines))

	for i, k := range klines {

		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
		volumes[i] = k.Volume
	}

	return highs, lows, closes, volumes
}

// 计算支撑位和阻力位
func CalculateSupportResistance(klines []Kline) *SupportResistance {
	highs, lows, closes, volumes := ConvertKlinesToPrices(klines)
	currentPrice := closes[len(closes)-1]

	result := &SupportResistance{
		CurrentPrice: currentPrice,
		Timestamp:    time.Now().Unix(),
	}

	// 1. 通过局部高点和低点识别
	pivotHighs := findPivotHighs(highs, 3)
	pivotLows := findPivotLows(lows, 3)

	// 2. 通过成交量加权价格区域识别
	volumeLevels := findVolumePriceLevels(highs, lows, volumes, 20)

	// 3. 合并所有水平并计算强度
	result.Resistances = mergeAndScoreLevels(pivotHighs, volumeLevels, "resistance", currentPrice)
	result.Supports = mergeAndScoreLevels(pivotLows, volumeLevels, "support", currentPrice)

	// 4. 过滤和排序
	result.Resistances = filterAndSortLevels(result.Resistances, currentPrice, true)
	result.Supports = filterAndSortLevels(result.Supports, currentPrice, false)

	return result
}

// 查找局部高点
func findPivotHighs(highs []float64, window int) []PriceLevel {
	var pivotHighs []PriceLevel

	for i := window; i < len(highs)-window; i++ {
		isPivot := true
		currentHigh := highs[i]

		// 检查是否是窗口内的最高点
		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if highs[j] > currentHigh {
				isPivot = false
				break
			}
		}

		if isPivot {
			// 计算强度基于窗口大小和价格变化
			strength := calculatePivotStrength(highs, i, window, true)
			pivotHighs = append(pivotHighs, PriceLevel{
				Price:    currentHigh,
				Strength: strength,
				Type:     "resistance",
				Touches:  1,
			})
		}
	}

	return pivotHighs
}

// 查找局部低点
func findPivotLows(lows []float64, window int) []PriceLevel {
	var pivotLows []PriceLevel

	for i := window; i < len(lows)-window; i++ {
		isPivot := true
		currentLow := lows[i]

		// 检查是否是窗口内的最低点
		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if lows[j] < currentLow {
				isPivot = false
				break
			}
		}

		if isPivot {
			strength := calculatePivotStrength(lows, i, window, false)
			pivotLows = append(pivotLows, PriceLevel{
				Price:    currentLow,
				Strength: strength,
				Type:     "support",
				Touches:  1,
			})
		}
	}

	return pivotLows
}

// 计算枢轴点强度
func calculatePivotStrength(prices []float64, index, window int, isHigh bool) float64 {
	strength := 0.0

	// 基于价格变化幅度
	priceChange := 0.0
	if isHigh {
		leftMin := findMin(prices[index-window : index])
		rightMin := findMin(prices[index+1 : index+window+1])
		priceChange = (prices[index] - leftMin + prices[index] - rightMin) / 2
	} else {
		leftMax := findMax(prices[index-window : index])
		rightMax := findMax(prices[index+1 : index+window+1])
		priceChange = (leftMax - prices[index] + rightMax - prices[index]) / 2
	}

	// 标准化强度
	avgPrice := (findMax(prices) + findMin(prices)) / 2
	if avgPrice > 0 {
		strength = priceChange / avgPrice
	}

	// 限制在0-1范围内
	if strength > 1 {
		strength = 1
	}
	if strength < 0.1 {
		strength = 0.1
	}

	return strength
}

// 基于成交量识别价格水平
func findVolumePriceLevels(highs, lows, volumes []float64, numLevels int) []PriceLevel {
	// 创建价格区间
	minPrice := findMin(lows)
	maxPrice := findMax(highs)
	rangeSize := (maxPrice - minPrice) / float64(numLevels)

	levels := make([]PriceLevel, numLevels)

	for i := 0; i < numLevels; i++ {
		levelMin := minPrice + float64(i)*rangeSize
		levelMax := levelMin + rangeSize

		// 计算该价格区间的总成交量
		totalVolume := 0.0
		for j := 0; j < len(highs); j++ {
			if (highs[j] >= levelMin && highs[j] <= levelMax) ||
				(lows[j] >= levelMin && lows[j] <= levelMax) {
				totalVolume += volumes[j]
			}
		}

		levels[i] = PriceLevel{
			Price:    (levelMin + levelMax) / 2,
			Strength: totalVolume / findMax(volumes), // 标准化强度
			Volume:   totalVolume,
		}
	}

	return levels
}

// 合并水平并计算分数
func mergeAndScoreLevels(pivots, volumeLevels []PriceLevel, levelType string, currentPrice float64) []PriceLevel {
	merged := make([]PriceLevel, 0)

	// 添加枢轴点
	merged = append(merged, pivots...)

	// 添加高成交量区域
	for _, level := range volumeLevels {
		if level.Strength > 0.3 { // 只添加高成交量区域
			level.Type = levelType
			merged = append(merged, level)
		}
	}

	// 合并相近的水平
	return mergeCloseLevels(merged, currentPrice*0.002) // 0.2% 范围内合并
}

// 合并相近的价格水平
func mergeCloseLevels(levels []PriceLevel, threshold float64) []PriceLevel {
	if len(levels) == 0 {
		return levels
	}

	// 按价格排序
	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price < levels[j].Price
	})

	merged := []PriceLevel{levels[0]}

	for i := 1; i < len(levels); i++ {
		last := &merged[len(merged)-1]
		current := levels[i]

		if current.Price-last.Price <= threshold {
			// 合并相近水平
			last.Price = (last.Price + current.Price) / 2
			last.Strength = (last.Strength + current.Strength) / 2
			last.Volume += current.Volume
			last.Touches += current.Touches
		} else {
			merged = append(merged, current)
		}
	}

	return merged
}

// 过滤和排序水平
func filterAndSortLevels(levels []PriceLevel, currentPrice float64, isResistance bool) []PriceLevel {
	filtered := make([]PriceLevel, 0)

	for _, level := range levels {
		// 过滤掉太接近当前价格的水平
		priceDiff := (level.Price - currentPrice) / currentPrice
		if isResistance && priceDiff > 0.01 || // 阻力位要高于当前价格1%
			!isResistance && priceDiff < -0.01 { // 支撑位要低于当前价格1%

			// 只保留强度足够的水平
			if level.Strength > 0.2 {
				filtered = append(filtered, level)
			}
		}
	}

	// 按强度排序
	sort.Slice(filtered, func(i, j int) bool {
		if isResistance {
			return filtered[i].Price < filtered[j].Price
		}
		return filtered[i].Price > filtered[j].Price
	})

	// 返回前5个最重要的水平
	if len(filtered) > 5 {
		return filtered[:5]
	}
	return filtered
}

// 工具函数
func findMin(arr []float64) float64 {
	min := arr[0]
	for _, v := range arr {
		if v < min {
			min = v
		}
	}
	return min
}

func findMax(arr []float64) float64 {
	max := arr[0]
	for _, v := range arr {
		if v > max {
			max = v
		}
	}
	return max
}

// 获取BTC支撑阻力位
func GetBTCAnalysis(klines []Kline) (*SupportResistance, error) {

	analysis := CalculateSupportResistance(klines)
	return analysis, nil
}

// 打印分析结果
func PrintAnalysis(analysis *SupportResistance) {
	fmt.Printf("=== BTC支撑阻力分析 ===\n")
	fmt.Printf("当前价格: $%.2f\n", analysis.CurrentPrice)
	fmt.Printf("分析时间: %s\n", time.Unix(analysis.Timestamp, 0).Format("2006-01-02 15:04:05"))

	fmt.Printf("\n🔺 阻力位:\n")
	for i, level := range analysis.Resistances {
		diffPercent := (level.Price - analysis.CurrentPrice) / analysis.CurrentPrice * 100
		fmt.Printf("%d. $%.2f (强度: %.1f%%, 距离: +%.1f%%)\n",
			i+1, level.Price, level.Strength*100, diffPercent)
	}

	fmt.Printf("\n🔻 支撑位:\n")
	for i, level := range analysis.Supports {
		diffPercent := (level.Price - analysis.CurrentPrice) / analysis.CurrentPrice * 100
		fmt.Printf("%d. $%.2f (强度: %.1f%%, 距离: %.1f%%)\n",
			i+1, level.Price, level.Strength*100, diffPercent)
	}
}
