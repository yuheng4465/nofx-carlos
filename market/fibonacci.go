package market

import "math"

// FibonacciLevels 斐波那契水平
type FibonacciLevels struct {
	RetracementLevels map[string]float64 // 回撤位
	ExtensionLevels   map[string]float64 // 扩展位
}

// PricePoint 价格点
type PricePoint struct {
	High float64
	Low  float64
}

// FibonacciAnalyzer 斐波那契分析器
type FibonacciAnalyzer struct {
	Levels FibonacciLevels
}

// NewFibonacciAnalyzer 创建斐波那契分析器
func NewFibonacciAnalyzer() *FibonacciAnalyzer {
	return &FibonacciAnalyzer{
		Levels: FibonacciLevels{
			RetracementLevels: map[string]float64{
				"0.236": 0.236,
				"0.382": 0.382,
				"0.500": 0.500,
				"0.618": 0.618,
				"0.786": 0.786,
			},
			ExtensionLevels: map[string]float64{
				"1.272": 1.272,
				"1.414": 1.414,
				"1.618": 1.618,
				"2.000": 2.000,
				"2.618": 2.618,
			},
		},
	}
}

// CalculateRetracement 计算斐波那契回撤位
func (fa *FibonacciAnalyzer) CalculateRetracement(high, low float64, trend string) map[string]float64 {
	levels := make(map[string]float64)
	distance := math.Abs(high - low)

	for name, ratio := range fa.Levels.RetracementLevels {
		var level float64
		if trend == "uptrend" {
			// 上升趋势：从高点回撤
			level = high - (distance * ratio)
		} else {
			// 下降趋势：从低点反弹
			level = low + (distance * ratio)
		}
		levels[name] = level
	}

	return levels
}

// CalculateExtension 计算斐波那契扩展位
func (fa *FibonacciAnalyzer) CalculateExtension(high, low float64, trend string) map[string]float64 {
	levels := make(map[string]float64)
	distance := math.Abs(high - low)

	for name, ratio := range fa.Levels.ExtensionLevels {
		var level float64
		if trend == "uptrend" {
			// 上升趋势扩展目标
			level = high + (distance * ratio)
		} else {
			// 下降趋势扩展目标
			level = low - (distance * ratio)
		}
		levels[name] = level
	}

	return levels
}

// FindSwingPoints 寻找摆动点（高点和低点）
func (fa *FibonacciAnalyzer) FindSwingPoints(prices []float64, window int) ([]PricePoint, string) {
	if len(prices) < window*2 {
		return nil, "insufficient_data"
	}

	var swingPoints []PricePoint
	var trend string

	// 分析趋势方向
	firstHalfAvg := average(prices[:len(prices)/2])
	secondHalfAvg := average(prices[len(prices)/2:])

	if secondHalfAvg > firstHalfAvg {
		trend = "uptrend"
	} else {
		trend = "downtrend"
	}

	// 寻找局部高点和低点
	for i := window; i < len(prices)-window; i++ {
		isHigh := true
		isLow := true

		// 检查是否为局部高点
		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if prices[j] > prices[i] {
				isHigh = false
			}
			if prices[j] < prices[i] {
				isLow = false
			}
		}

		if isHigh {
			swingPoints = append(swingPoints, PricePoint{High: prices[i], Low: 0})
		} else if isLow {
			swingPoints = append(swingPoints, PricePoint{High: 0, Low: prices[i]})
		}
	}

	return swingPoints, trend
}

// FibonacciTrendAnalysis 斐波那契趋势分析
func (fa *FibonacciAnalyzer) FibonacciTrendAnalysis(prices []float64) map[string]interface{} {
	result := make(map[string]interface{})

	// 寻找关键摆动点
	swingPoints, trend := fa.FindSwingPoints(prices, 5)
	if len(swingPoints) < 2 {
		result["error"] = "不足的摆动点数据"
		return result
	}

	// 提取显著的高点和低点
	var significantHighs, significantLows []float64
	for _, point := range swingPoints {
		if point.High > 0 {
			significantHighs = append(significantHighs, point.High)
		}
		if point.Low > 0 {
			significantLows = append(significantLows, point.Low)
		}
	}

	if len(significantHighs) == 0 || len(significantLows) == 0 {
		result["error"] = "无法确定关键价格水平"
		return result
	}

	// 确定主要的高点和低点
	mainHigh := max_val(significantHighs)
	mainLow := min_val(significantLows)

	result["trend"] = trend
	result["main_high"] = mainHigh
	result["main_low"] = mainLow

	// 计算斐波那契水平
	result["retracement_levels"] = fa.CalculateRetracement(mainHigh, mainLow, trend)
	result["extension_levels"] = fa.CalculateExtension(mainHigh, mainLow, trend)

	// 当前价格分析
	currentPrice := prices[len(prices)-1]
	result["current_price"] = currentPrice

	// 分析当前价格相对于斐波那契水平的位置
	retracementLevels := result["retracement_levels"].(map[string]float64)
	extensionLevels := result["extension_levels"].(map[string]float64)

	result["price_analysis"] = fa.analyzePricePosition(currentPrice, retracementLevels, extensionLevels, trend)

	return result
}

// analyzePricePosition 分析当前价格位置
func (fa *FibonacciAnalyzer) analyzePricePosition(currentPrice float64, retracement, extension map[string]float64, trend string) map[string]interface{} {
	analysis := make(map[string]interface{})

	// 找到最近的回撤位
	closestRetracement := ""
	minDistance := math.MaxFloat64

	for level, price := range retracement {
		distance := math.Abs(currentPrice - price)
		if distance < minDistance {
			minDistance = distance
			closestRetracement = level
		}
	}

	analysis["closest_retracement"] = closestRetracement
	analysis["distance_to_retracement"] = minDistance

	// 判断价格行为
	if trend == "uptrend" {
		if currentPrice > retracement["0.618"] {
			analysis["momentum"] = "strong"
		} else if currentPrice > retracement["0.382"] {
			analysis["momentum"] = "moderate"
		} else {
			analysis["momentum"] = "weak"
		}
	} else {
		if currentPrice < retracement["0.618"] {
			analysis["momentum"] = "strong"
		} else if currentPrice < retracement["0.382"] {
			analysis["momentum"] = "moderate"
		} else {
			analysis["momentum"] = "weak"
		}
	}

	return analysis
}

// FibonacciTimeZones 斐波那契时间周期分析
func (fa *FibonacciAnalyzer) FibonacciTimeZones(prices []float64, startIndex int) []int {
	fibNumbers := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}
	var timeZones []int

	for _, fib := range fibNumbers {
		if startIndex+fib < len(prices) {
			timeZones = append(timeZones, startIndex+fib)
		}
	}

	return timeZones
}

// 辅助函数
func average(numbers []float64) float64 {
	sum := 0.0
	for _, num := range numbers {
		sum += num
	}
	return sum / float64(len(numbers))
}

func max_val(numbers []float64) float64 {
	max := numbers[0]
	for _, num := range numbers {
		if num > max {
			max = num
		}
	}
	return max
}

func min_val(numbers []float64) float64 {
	min := numbers[0]
	for _, num := range numbers {
		if num < min {
			min = num
		}
	}
	return min
}
