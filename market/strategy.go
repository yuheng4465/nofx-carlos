package market

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Knetic/govaluate"
)

// 配置结构体定义
type StrategyEngine struct {
	Name       string                `json:"name"`
	Version    float64               `json:"version"`
	Strategies map[string][]Strategy `json:"strategies"`
}

type Strategy struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Side        string   `json:"side"`
	Confidence  float64  `json:"confidence"`
	Expression  string   `json:"expression"`
	Fields      []string `json:"fields"`
}

// 决策执行结果
type StrategyResult struct {
	Matched         bool     `json:"matched"`
	MatchedIndex    int      `json:"matchedIndex"`
	MatchedStrategy Strategy `json:"matchedStrategy"`
	Actions         []string `json:"actions"`
}

type Cases struct {
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
type AdvancedExpressionEvaluator struct{}

func NewAdvancedExpressionEvaluator() *AdvancedExpressionEvaluator {
	return &AdvancedExpressionEvaluator{}
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

	log.Printf("配置加载成功，版本: %.1f, 策略数量: %d",
		cl.Engine.Version, len(cl.Engine.Strategies))
	return nil
}

// 验证字段是否存在
func (de *DecisionEngine) validateFields(requiredFields []string, userData *Cases) error {
	for _, field := range requiredFields {
		if _, exists := userData.Metrics[field]; !exists {
			return fmt.Errorf("缺少必需字段: %s", field)
		}
	}
	return nil
}

// 执行决策
func (de *DecisionEngine) ExecuteStrategy(target string, userData *Cases) (StrategyResult, error) {
	var strategyResult StrategyResult
	strategyResult.Matched = false
	_, exists := de.configLoader.Engine.Strategies[target]
	if !exists {
		return strategyResult, fmt.Errorf("策略不存在: %s", target)
	}
	for i, strategy := range de.configLoader.Engine.Strategies[target] {
		// 策略未启用
		if !strategy.Enabled {
			continue
		}

		strategyResult.MatchedIndex = i
		strategyResult.MatchedStrategy = strategy
		// 验证必需字段
		if err := de.validateFields(strategy.Fields, userData); err != nil {
			return strategyResult, err
		}

		// 执行表达式求值
		result, err := de.evaluator.Evaluate(strategy.Expression, userData)
		if err != nil {
			return strategyResult, err
		}

		// 如果条件满足，返回动作列表
		if result {
			strategyResult.Matched = true
			return strategyResult, nil
		}
	}

	return strategyResult, fmt.Errorf("策略未触发: %s", target)
}

func (aee *AdvancedExpressionEvaluator) Evaluate(expression string, userData *Cases) (bool, error) {
	// 创建表达式
	expr, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return false, fmt.Errorf("表达式解析失败: %v", err)
	}

	// 准备参数（移除 user. 前缀）
	parameters := make(map[string]interface{})
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
