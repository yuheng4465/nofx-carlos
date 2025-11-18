package market

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/Knetic/govaluate"
)

type Logical string

const (
	AND Logical = "and"
	OR  Logical = "or"
)

// 配置结构体定义
type StrategyEngine struct {
	Name       string     `json:"name"`
	Version    float64    `json:"version"`
	Strategies []Strategy `json:"strategies"`
}

type Strategy struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Side        string        `json:"side"`
	Confidence  float64       `json:"confidence"`
	Group       StrategyGroup `json:"group"`
}

type StrategyGroup struct {
	Expressions []StrategyExpression `json:"expressions"`
	Logical     string               `json:"logical"`
}

type StrategyExpression struct {
	Expression  string   `json:"expression"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"`
}

// 决策执行结果
type StrategyResult struct {
	Matched         bool     `json:"matched"`
	MatchedStrategy Strategy `json:"matchedStrategy"`
}

type StrategyData struct {
	Name    string                 `json:"name"`
	Metrics map[string]interface{} `json:"metrics"`
}

// 决策引擎
type DecisionEngine struct {
	configLoader *ConfigLoader
	evaluator    *AdvancedExpressionEvaluator
}

// 配置加载器
type ConfigLoader struct {
	Engine *StrategyEngine
}

// 使用 govaluate 的高级表达式求值器
type AdvancedExpressionEvaluator struct {
	functions map[string]govaluate.ExpressionFunction
}

func NewAdvancedExpressionEvaluator() *AdvancedExpressionEvaluator {
	return &AdvancedExpressionEvaluator{
		functions: createFunctions(),
	}
}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		Engine: &StrategyEngine{},
	}
}

func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{
		configLoader: NewConfigLoader(),
		evaluator:    NewAdvancedExpressionEvaluator(),
	}
}

// 创建自定义函数映射
func createFunctions() map[string]govaluate.ExpressionFunction {
	functions := make(map[string]govaluate.ExpressionFunction)

	// 数学函数
	functions["abs"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("abs function requires exactly one argument")
		}
		switch v := args[0].(type) {
		case float64:
			return math.Abs(v), nil
		case int:
			return math.Abs(float64(v)), nil
		default:
			return nil, fmt.Errorf("abs function requires numeric argument, got %T", v)
		}
	}

	functions["max"] = func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("max function requires exactly two arguments")
		}
		result, err := toFloat64(args[0], args[1])
		if err != nil {
			return nil, err
		}
		return math.Max(result[0], result[1]), nil
	}

	functions["min"] = func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("min function requires exactly two arguments")
		}
		result, err := toFloat64(args[0], args[1])
		if err != nil {
			return nil, err
		}
		return math.Min(result[0], result[1]), nil
	}

	functions["sqrt"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sqrt function requires exactly one argument")
		}
		result, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		val := result[0]
		if val < 0.0 {
			return nil, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(val), nil
	}

	functions["pow"] = func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("pow function requires exactly two arguments")
		}
		result, err := toFloat64(args[0], args[1])
		if err != nil {
			return nil, err
		}
		return math.Pow(result[0], result[1]), nil
	}

	functions["exp"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("exp function requires exactly one argument")
		}
		result, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return math.Exp(result[0]), nil
	}

	functions["log"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("log function requires exactly one argument")
		}
		result, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		if result[0] <= 0 {
			return nil, fmt.Errorf("log of non-positive number")
		}
		return math.Log(result[0]), nil
	}

	functions["log10"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("log10 function requires exactly one argument")
		}
		result, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		if result[0] <= 0 {
			return nil, fmt.Errorf("log10 of non-positive number")
		}
		return math.Log10(result[0]), nil
	}

	// 统计函数
	functions["avg"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("avg function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("avg() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("avg() 参数必须是数组类型")
		}

		sum := 0.0
		for _, arg := range values {
			result, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			sum += result[0]
		}
		return sum / float64(len(args)), nil
	}

	// 取序列最后一个元素(当前元素)
	functions["current"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("last function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		default:
			return 0.0, fmt.Errorf("current() 参数必须是数组类型")
		}
		// if len(values) == 0 {
		// 	return 0.0, fmt.Errorf("current() 数组为空")
		// }
		return values[len(values)-1], nil
	}

	// 取序列上一个元素
	functions["prev"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("prev function requires at least one argument")
		}
		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		default:
			return 0.0, fmt.Errorf("prev() 参数必须是数组类型")
		}
		return values[len(values)-2], nil
	}

	// 取序列高点
	functions["high"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("high function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("high() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("high() 参数必须是数组类型")
		}

		high := 0.0
		for _, arg := range values {
			result, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			if high < result[0] {
				high = result[0]
			}
		}
		return high, nil
	}

	// 取序列低点
	functions["low"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("low function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("low() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("low() 参数必须是数组类型")
		}

		low := 0.0
		for _, arg := range values {
			result, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			if low > result[0] {
				low = result[0]
			}
		}
		return low, nil
	}

	// 取序列高点下标
	functions["highIndex"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("highIndex function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("highIndex() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("highIndex() 参数必须是数组类型")
		}

		highIndex := 0
		high := 0.0
		for i, arg := range values {
			result, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				high = result[0]
			} else {
				if high < result[0] {
					high = result[0]
					highIndex = i
				}
			}
		}
		return highIndex, nil
	}

	// 取序列低点下标
	functions["lowIndex"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("lowIndex function requires at least one argument")
		}

		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("lowIndex() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("lowIndex() 参数必须是数组类型")
		}

		lowIndex := 0
		low := 0.0
		for i, arg := range values {
			result, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				low = result[0]
			} else {
				if low > result[0] {
					low = result[0]
					lowIndex = i
				}
			}
		}
		return lowIndex, nil
	}

	// 取序列线性斜率
	functions["slope"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("slope function requires at least one argument")
		}
		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("slope() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("slope() 参数必须是数组类型")
		}

		if len(values) < 2 {
			return 0.0, nil
		}
		// 使用线性回归计算斜率
		n := len(values)
		var sumX, sumY, sumXY, sumX2 float64

		for i, y := range values {
			x := float64(i)
			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}

		// 斜率公式: (n*Σxy - Σx*Σy) / (n*Σx² - (Σx)²)
		numerator := float64(n)*sumXY - sumX*sumY
		denominator := float64(n)*sumX2 - sumX*sumX

		if denominator == 0 {
			return 0.0, nil
		}
		return numerator / denominator, nil
	}

	// 取序列线性斜率强度
	functions["rSquared"] = func(args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("rSquared function requires at least one argument")
		}
		var values []float64
		switch v := args[0].(type) {
		case []float64:
			values = v
		case []any:
			values = make([]float64, len(v))
			for i, item := range v {
				if f, ok := item.(float64); ok {
					values[i] = f
				} else {
					return 0.0, fmt.Errorf("rSquared() 数组元素必须是数字")
				}
			}
		default:
			return 0.0, fmt.Errorf("rSquared() 参数必须是数组类型")
		}

		n := float64(len(values))
		var sumX, sumY, sumXY, sumXX float64

		for i, price := range values {
			x := float64(i)
			y := price
			sumX += x
			sumY += y
			sumXY += x * y
			sumXX += x * x
		}

		// 计算斜率
		slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

		// 计算趋势强度 (R-squared)
		meanY := sumY / n
		var totalSS, regSS float64
		for i, price := range values {
			x := float64(i)
			predicted := (slope * x) + (sumY/n - slope*sumX/n)
			totalSS += math.Pow(price-meanY, 2)
			regSS += math.Pow(predicted-meanY, 2)
		}

		rSquared := 0.0
		if totalSS > 0 {
			rSquared = regSS / totalSS
		}
		return rSquared, nil
	}

	// 判断是否出现新高
	functions["newHigh"] = func(args ...any) (any, error) {
		if len(args) <= 1 {
			return false, fmt.Errorf("newHigh function requires at least two argument")
		}

		var data []float64
		removing := 0
		if floatSlice, ok := args[0].([]float64); ok {
			data = floatSlice
		} else {
			return false, fmt.Errorf("newHigh function argument1 type is not []float64")
		}

		removing, err := interfaceToInt(args[1])
		if err != nil {
			return false, fmt.Errorf("newHigh function argument2 type is not int")
		}

		if len(data) < removing {
			return false, nil
		}

		highIndex := 0
		high := 0.0
		len := 0
		for i, arg := range data {
			result, err := toFloat64(arg)
			if err != nil {
				return false, err
			}
			if i == 0 {
				high = result[0]
			} else {
				if high < result[0] {
					high = result[0]
					highIndex = i
				}
			}
			len++
		}

		if len < removing {
			return false, nil
		}

		isNewHigh := false
		if highIndex > len-removing {
			isNewHigh = true
		}

		return isNewHigh, nil
	}

	// 判断是否出现新低
	functions["newLow"] = func(args ...any) (any, error) {
		if len(args) <= 1 {
			return false, fmt.Errorf("newLow function requires at least two argument")
		}

		var data []float64
		removing := 0
		if floatSlice, ok := args[0].([]float64); ok {
			data = floatSlice
		} else {
			return false, fmt.Errorf("newLow function argument1 type is not []float64")
		}

		removing, err := interfaceToInt(args[1])
		if err != nil {
			return false, fmt.Errorf("newLow function argument2 type is not int")
		}

		if len(data) < removing {
			return false, nil
		}

		lowIndex := 0
		low := 0.0
		len := 0
		for i, arg := range data {
			result, err := toFloat64(arg)
			if err != nil {
				return false, err
			}
			if i == 0 {
				low = result[0]
			} else {
				if low > result[0] {
					low = result[0]
					lowIndex = i
				}
			}
			len++
		}

		if len < removing {
			return false, nil
		}

		isNewHigh := false
		if lowIndex > len-removing {
			isNewHigh = true
		}

		return isNewHigh, nil
	}

	return functions
}

func interfaceToInt(i interface{}) (int, error) {
	switch v := i.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("不支持的类型: %T", i)
	}
}

// 辅助函数：将接口转换为float64
func toFloat64(values ...any) ([]float64, error) {
	result := make([]float64, len(values))
	for i, v := range values {
		switch val := v.(type) {
		case float64:
			result[i] = val
		case int:
			result[i] = float64(val)
		case float32:
			result[i] = float64(val)
		default:
			return nil, fmt.Errorf("cannot convert %T to float64", v)
		}
	}
	return result, nil
}

// 加载配置
func (de *DecisionEngine) LoadConfig(filename string) error {
	return de.configLoader.LoadFromFile(filename)
}

// 从文件加载配置
func (cl *ConfigLoader) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	return cl.LoadFromJSON(data)
}

// 从JSON字符串加载配置
func (cl *ConfigLoader) LoadFromJSON(data []byte) error {
	if err := json.Unmarshal(data, &cl.Engine); err != nil {
		return fmt.Errorf("解析JSON配置失败: %v", err)
	}

	log.Printf("配置加载成功，版本: %.1f, 策略组数量: %d",
		cl.Engine.Version, len(cl.Engine.Strategies))
	return nil
}

// 验证字段是否存在
func (de *DecisionEngine) validateFields(requiredFields []string, userData *StrategyData) error {
	for _, field := range requiredFields {
		if _, exists := userData.Metrics[field]; !exists {
			return fmt.Errorf("缺少必需字段: %s", field)
		}
	}
	return nil
}

// 获取有效策略并且根据评分排序
func (de *DecisionEngine) getStrategies() ([]Strategy, error) {
	if len(de.configLoader.Engine.Strategies) == 0 {
		return nil, fmt.Errorf("策略不存在")
	}

	var strategies []Strategy
	for _, strategy := range de.configLoader.Engine.Strategies {
		if strategy.Enabled {
			strategies = append(strategies, strategy)
		}
	}

	if len(strategies) == 0 {
		return nil, fmt.Errorf("未有生效策略")
	}

	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Confidence > strategies[j].Confidence
	})

	return strategies, nil
}

// 执行and策略
func (de *DecisionEngine) ExecuteAnd(group StrategyGroup, userData *StrategyData) (bool, error) {

	for _, expression := range group.Expressions {
		// 验证必需字段
		if err := de.validateFields(expression.Fields, userData); err != nil {
			return false, err
		}

		// 执行表达式求值
		result, err := de.evaluator.Evaluate(expression.Expression, userData)
		if err != nil {
			return false, err
		}

		if !result {
			return false, nil
		}
	}

	return true, nil
}

// 执行or策略
func (de *DecisionEngine) ExecuteOr(group StrategyGroup, userData *StrategyData) (bool, error) {

	for _, expression := range group.Expressions {
		// 验证必需字段
		if err := de.validateFields(expression.Fields, userData); err != nil {
			return false, err
		}

		// 执行表达式求值
		result, err := de.evaluator.Evaluate(expression.Expression, userData)
		if err != nil {
			return false, err
		}

		if result {
			return true, nil
		}
	}

	return false, nil
}

// 执行决策
func (de *DecisionEngine) ExecuteStrategy(userData *StrategyData) (StrategyResult, error) {
	var strategyResult StrategyResult
	strategyResult.Matched = false

	// 获取策略
	strategies, err := de.getStrategies()
	if err != nil {
		return strategyResult, err
	}

	for _, strategy := range strategies {
		strategyResult.MatchedStrategy = strategy
		if strategy.Group.Logical == string(AND) {
			result, err := de.ExecuteAnd(strategy.Group, userData)
			if err != nil {
				return strategyResult, err
			}

			// 如果条件满足，返回动作列表
			if result {
				strategyResult.Matched = true
				return strategyResult, nil
			}
		}

		if strategy.Group.Logical == string(OR) {
			result, err := de.ExecuteOr(strategy.Group, userData)
			if err != nil {
				return strategyResult, err
			}

			// 如果条件满足，返回动作列表
			if result {
				strategyResult.Matched = true
				return strategyResult, nil
			}
		}
	}

	return strategyResult, fmt.Errorf("策略未触发")
}

// 执行
func (aee *AdvancedExpressionEvaluator) Evaluate(expression string, userData *StrategyData) (bool, error) {
	// 创建带函数的表达式
	expr, err := govaluate.NewEvaluableExpressionWithFunctions(expression, aee.functions)
	if err != nil {
		return false, fmt.Errorf("failed to parse expression: %v", err)
	}

	// 准备参数（移除 user. 前缀）
	parameters := make(map[string]any)
	for key, value := range userData.Metrics {
		parameters[key] = value
	}

	// 执行求值
	result, err := expr.Evaluate(parameters)
	if err != nil {
		return false, fmt.Errorf("表达式求值失败: %v", err)
	}

	// 转换为布尔值
	if boolResult, ok := result.(bool); ok {
		return boolResult, nil
	}

	return false, fmt.Errorf("表达式结果不是布尔类型: %v", result)
}
