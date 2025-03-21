/**
  @author: Hanhai
  @since: 2025/3/16 20:44:20
  @desc: 模型特定的密钥选择策略
**/

package key

import (
	"flowsilicon/internal/config"
	"flowsilicon/internal/logger"
	"strings"
	"time"
)

// GetModelSpecificKey 根据模型名称获取特定的密钥
func GetModelSpecificKey(modelName string) (string, bool, error) {
	// 检查是否有针对该模型的特定策略配置
	cfg := config.GetConfig()

	// 添加调试日志
	logger.Info("检查模型特定策略: 模型=%s", modelName)
	logger.Info("当前配置的模型策略列表: %v", cfg.App.ModelKeyStrategies)

	// 直接查找精确匹配
	if strategyID, exists := cfg.App.ModelKeyStrategies[modelName]; exists {
		// 记录找到的策略
		logger.Info("找到模型特定策略(精确匹配): 模型=%s, 策略ID=%d", modelName, strategyID)
		return applyModelStrategy(modelName, strategyID)
	}

	// 如果精确匹配失败，尝试不区分大小写的匹配
	modelNameLower := strings.ToLower(modelName)
	for configModel, strategyID := range cfg.App.ModelKeyStrategies {
		if strings.ToLower(configModel) == modelNameLower {
			// 记录找到的策略
			logger.Info("找到模型特定策略(不区分大小写): 模型=%s 匹配配置=%s, 策略ID=%d",
				modelName, configModel, strategyID)
			return applyModelStrategy(modelName, strategyID)
		}
	}

	// 没有找到特定策略
	logger.Info("未找到模型特定策略: 模型=%s", modelName)
	return "", false, nil
}

// applyModelStrategy 应用模型特定策略
func applyModelStrategy(modelName string, strategyID int) (string, bool, error) {
	switch strategyID {
	case 1: // 高成功率策略
		logger.Info("使用高成功率策略选择密钥: 模型=%s", modelName)
		key, err := getHighSuccessRateKey(modelName)
		return key, true, err
	case 2: // 高分数策略
		logger.Info("使用高分数策略选择密钥: 模型=%s", modelName)
		key, err := GetOptimalApiKey()
		return key, true, err
	case 3: // 低RPM策略
		logger.Info("使用低RPM策略选择密钥: 模型=%s", modelName)
		key, err := getFastResponseKey()
		return key, true, err
	case 4: // 低TPM策略
		logger.Info("使用低TPM策略选择密钥: 模型=%s", modelName)
		activeKeys := config.GetActiveApiKeys()
		if len(activeKeys) == 0 {
			return "", true, ErrNoActiveKeys
		}

		var bestKey string
		var lowestTPM int = 999999

		for _, key := range activeKeys {
			if key.Balance < config.GetConfig().App.MinBalanceThreshold {
				continue
			}

			if key.TokensPerMinute < lowestTPM {
				lowestTPM = key.TokensPerMinute
				bestKey = key.Key
			}
		}

		if bestKey == "" {
			key, err := getAnyAvailableKey()
			return key, true, err
		}

		config.UpdateApiKeyLastUsed(bestKey, time.Now().Unix())
		return bestKey, true, nil
	case 5: // 高余额策略
		logger.Info("使用高余额策略选择密钥: 模型=%s", modelName)
		key, err := getHighestBalanceKey()
		return key, true, err
	default:
		logger.Info("使用默认策略(高分数)选择密钥: 模型=%s", modelName)
		key, err := GetOptimalApiKey()
		return key, true, err
	}
}
