/**
  @author: Hanhai
  @since: 2025/3/27 00:12:20
  @desc:
**/

package model

import (
	"database/sql"
	"encoding/json"
	"flowsilicon/internal/config"
	"flowsilicon/internal/logger"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	// 数据库实例
	modelDB *sql.DB
)

// InitModelDB 初始化模型数据库
func InitModelDB(dbPath string) error {
	var err error
	modelDB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// 测试数据库连接
	if err = modelDB.Ping(); err != nil {
		return err
	}

	// 创建模型表
	query := `CREATE TABLE IF NOT EXISTS models (
		id TEXT PRIMARY KEY,
		is_free BOOLEAN DEFAULT 0 NOT NULL,
		is_giftable BOOLEAN DEFAULT 0 NOT NULL,
		strategy_id INTEGER DEFAULT 0 NOT NULL,
		type INTEGER DEFAULT 1 NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	)`
	_, err = modelDB.Exec(query)
	if err != nil {
		logger.Error("创建模型表失败: %v", err)
		return err
	}

	// 检查表是否确实创建成功
	var tableExists int
	err = modelDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='models'").Scan(&tableExists)
	if err != nil {
		logger.Error("检查模型表存在失败: %v", err)
		return err
	}

	if tableExists == 0 {
		logger.Error("模型表未创建成功，请检查数据库权限")
		return fmt.Errorf("模型表未创建成功")
	}

	// 检查是否需要添加新字段
	var strategyColumnExists int
	err = modelDB.QueryRow("SELECT count(*) FROM pragma_table_info('models') WHERE name='strategy_id'").Scan(&strategyColumnExists)
	if err != nil {
		logger.Error("检查strategy_id字段存在失败: %v", err)
		return err
	}

	// 检查type字段是否存在
	var typeColumnExists int
	err = modelDB.QueryRow("SELECT count(*) FROM pragma_table_info('models') WHERE name='type'").Scan(&typeColumnExists)
	if err != nil {
		logger.Error("检查type字段存在失败: %v", err)
		return err
	}

	// 如果列不存在，添加它
	if strategyColumnExists == 0 {
		_, err = modelDB.Exec("ALTER TABLE models ADD COLUMN strategy_id INTEGER DEFAULT 0 NOT NULL")
		if err != nil {
			logger.Error("添加strategy_id字段失败: %v", err)
			return err
		}
		logger.Info("成功添加strategy_id字段到models表")
	}

	// 如果type列不存在，添加它
	if typeColumnExists == 0 {
		_, err = modelDB.Exec("ALTER TABLE models ADD COLUMN type INTEGER DEFAULT 1 NOT NULL")
		if err != nil {
			logger.Error("添加type字段失败: %v", err)
			return err
		}
		logger.Info("成功添加type字段到models表")
	}

	// 更新所有免费模型的策略为8（免费策略），默认策略为6（普通策略）
	_, err = modelDB.Exec(`UPDATE models SET 
							strategy_id = CASE 
								WHEN is_free = 1 AND (strategy_id = 0 OR strategy_id IS NULL) THEN 8 
								WHEN (strategy_id = 0 OR strategy_id IS NULL) THEN 6
								ELSE strategy_id 
							END, 
							updated_at = CURRENT_TIMESTAMP 
						  WHERE deleted_at IS NULL`)
	if err != nil {
		logger.Error("更新模型默认策略失败: %v", err)
		// 继续执行，因为这不是致命错误
	} else {
		logger.Info("已更新模型默认策略：免费模型使用策略8，其他模型使用策略6")
	}

	logger.Info("模型表初始化成功")
	return nil
}

// CloseModelDB 关闭模型数据库
func CloseModelDB() error {
	if modelDB != nil {
		return modelDB.Close()
	}
	return nil
}

// GetAllModels 获取所有模型
func GetAllModels() ([]Model, error) {
	// 确保数据库连接已经初始化
	if modelDB == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 检查是否存在可用的API密钥
	activeApiKeys := config.GetActiveApiKeys()
	if len(activeApiKeys) == 0 {
		return nil, fmt.Errorf("没有可用的API密钥,无法更新模型")
	}

	// 查询模型数量
	modelsCount_, err := GetModelsCount()
	if err != nil {
		return nil, err
	}

	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL

	if modelsCount_ == 0 {
		modelIds, _, err := fetchRemoteModels(baseURL)
		if err != nil {
			return nil, err
		}

		SaveModels(modelIds)
	}

	// 查询所有未删除的模型
	query := `SELECT id, is_free, is_giftable, strategy_id, type FROM models WHERE deleted_at IS NULL`
	rows, err := modelDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		var model Model
		if err := rows.Scan(&model.ID, &model.IsFree, &model.IsGiftable, &model.StrategyID, &model.Type); err != nil {
			return nil, err
		}
		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

// TODO 与web/handle.go中的fetchRemoteModels重复
// 从远程API获取模型列表
func fetchRemoteModels(baseURL string) ([]string, int, error) {

	// 构建API请求URL
	url := strings.TrimRight(baseURL, "/") + "/v1/models"

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	apikeys := config.GetActiveApiKeys()
	req.Header.Set("Authorization", "Bearer "+apikeys[0].Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, 0, err
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}

	// 提取模型列表
	data, ok := result["data"].([]interface{})
	if !ok {
		logger.Error("解析模型列表失败: data字段不是数组")
		return nil, 0, nil
	}

	// 提取模型ID
	var modelIds []string
	for _, item := range data {
		model, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id, ok := model["id"].(string)
		if !ok {
			continue
		}

		modelIds = append(modelIds, id)
	}

	return modelIds, len(modelIds), nil
}

// SaveModels 保存模型列表到数据库
// 对于API获取的模型列表，与数据库已有模型进行对比
// 保留库中存在且API中也存在的模型，删除库中存在但API中不存在的模型
// 添加库中不存在但API中存在的模型
func SaveModels(modelIds []string) (int, error) {
	// 确保数据库连接已经初始化
	if modelDB == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	// 开始事务
	tx, err := modelDB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 先将所有模型标记为已删除
	_, err = tx.Exec("UPDATE models SET deleted_at = CURRENT_TIMESTAMP WHERE deleted_at IS NULL")
	if err != nil {
		return 0, err
	}

	// 准备插入或更新模型的语句
	insertOrUpdate := `INSERT INTO models (id, is_free, is_giftable, strategy_id, type, deleted_at) 
						VALUES (?, ?, ?, ?, ?, NULL)
						ON CONFLICT(id) DO UPDATE SET 
						is_free = ?, 
						is_giftable = ?,
						deleted_at = NULL, 
						updated_at = CURRENT_TIMESTAMP`

	// 统计新增或更新的模型数量
	count := 0

	// 遍历API获取的模型ID
	for _, modelId := range modelIds {
		// 检查是否是免费模型
		isFree := isModelFree(modelId)
		// 检查是否可用赠费
		isGiftable := isModelGiftable(modelId)
		// 检查是否是推理模型
		isReason := isModelReason(modelId)

		// 设置策略ID：免费模型使用策略8（免费策略），非免费模型使用策略6（普通策略）
		var strategyID int = 6 // 默认为普通策略
		if isFree {
			strategyID = 8 // 免费模型使用免费策略
		}

		// 默认模型类型为1（对话）
		var modelType int = 1
		// 如果是推理模型，将类型设置为7
		if isReason {
			modelType = 7 // 推理模型类型为7
		}

		// 插入或更新模型
		_, err = tx.Exec(insertOrUpdate, modelId, isFree, isGiftable, strategyID, modelType, isFree, isGiftable)
		if err != nil {
			return 0, err
		}
		count++
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

// GetModelsCount 获取模型数量
func GetModelsCount() (int, error) {
	// 确保数据库连接已经初始化
	if modelDB == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	var count int
	err := modelDB.QueryRow("SELECT COUNT(*) FROM models WHERE deleted_at IS NULL").Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// isModelFree 检查模型是否免费
func isModelFree(modelId string) bool {
	for _, freeModel := range FreeModels {
		if freeModel == modelId {
			return true
		}
	}
	return false
}

// isModelGiftable 检查模型是否可用赠费
func isModelGiftable(modelId string) bool {
	for _, giftableModel := range GiftableModels {
		if giftableModel == modelId {
			return true
		}
	}
	return false
}

// isModelReason 检查模型是否是推理模型
func isModelReason(modelId string) bool {
	for _, reasonModel := range ReasonModels {
		if reasonModel == modelId {
			return true
		}
	}
	return false
}

// UpdateModelStrategy 更新模型策略
func UpdateModelStrategy(modelId string, strategyId int) error {
	if modelDB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 更新模型策略
	_, err := modelDB.Exec(
		"UPDATE models SET strategy_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		strategyId, modelId)
	if err != nil {
		logger.Error("更新模型策略失败: %v", err)
		return err
	}

	logger.Info("已更新模型 %s 的策略为 %d", modelId, strategyId)
	return nil
}

// GetModelStrategy 获取模型策略
func GetModelStrategy(modelId string) (int, error) {
	if modelDB == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	var strategyId int
	err := modelDB.QueryRow(
		"SELECT strategy_id FROM models WHERE id = ? AND deleted_at IS NULL",
		modelId).Scan(&strategyId)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // 未找到模型，返回默认策略0
		}
		logger.Error("获取模型策略失败: %v", err)
		return 0, err
	}

	return strategyId, nil
}

// UpdateModelType 更新模型类型
func UpdateModelType(modelId string, modelType int) error {
	if modelDB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 更新模型类型
	_, err := modelDB.Exec(
		"UPDATE models SET type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		modelType, modelId)
	if err != nil {
		logger.Error("更新模型类型失败: %v", err)
		return err
	}

	logger.Info("已更新模型 %s 的类型为 %d", modelId, modelType)
	return nil
}

// GetModelType 获取模型类型
func GetModelType(modelId string) (int, error) {
	if modelDB == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}

	var modelType int
	err := modelDB.QueryRow(
		"SELECT type FROM models WHERE id = ? AND deleted_at IS NULL",
		modelId).Scan(&modelType)
	if err != nil {
		if err == sql.ErrNoRows {
			return 1, nil // 未找到模型，返回默认类型1（对话）
		}
		logger.Error("获取模型类型失败: %v", err)
		return 0, err
	}

	return modelType, nil
}

// BeginTransaction 开始一个数据库事务
func BeginTransaction() (*sql.Tx, error) {
	if modelDB == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}
	return modelDB.Begin()
}

// DeleteModelStrategy 从数据库中删除模型策略记录
func DeleteModelStrategy(modelId string) error {
	if modelDB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 从数据库中删除模型策略记录
	_, err := modelDB.Exec(
		"DELETE FROM models WHERE id = ?",
		modelId)
	if err != nil {
		logger.Error("从数据库删除模型策略记录失败: %v", err)
		return err
	}

	logger.Info("已从数据库中删除模型 %s 的策略记录", modelId)
	return nil
}

// UpdateModelTypeWithTx 使用事务更新模型类型
func UpdateModelTypeWithTx(tx *sql.Tx, modelId string, modelType int) error {
	if tx == nil {
		return fmt.Errorf("事务对象为空")
	}

	// 更新模型类型
	_, err := tx.Exec(
		"UPDATE models SET type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		modelType, modelId)
	if err != nil {
		logger.Error("使用事务更新模型类型失败: %v", err)
		return err
	}

	logger.Info("已使用事务更新模型 %s 的类型为 %d", modelId, modelType)
	return nil
}

// UpdateModelStrategyWithTx 使用事务更新模型策略
func UpdateModelStrategyWithTx(tx *sql.Tx, modelId string, strategyId int) error {
	if tx == nil {
		return fmt.Errorf("事务对象为空")
	}

	// 更新模型策略
	_, err := tx.Exec(
		"UPDATE models SET strategy_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		strategyId, modelId)
	if err != nil {
		logger.Error("使用事务更新模型策略失败: %v", err)
		return err
	}

	logger.Info("已使用事务更新模型 %s 的策略为 %d", modelId, strategyId)
	return nil
}

// UpdateModelFreeStatusWithTx 使用事务更新模型免费状态
func UpdateModelFreeStatusWithTx(tx *sql.Tx, modelIds []string, isFree bool) (int, error) {
	if tx == nil {
		return 0, fmt.Errorf("事务对象为空")
	}

	if len(modelIds) == 0 {
		return 0, nil
	}

	// 构建IN查询参数
	placeholders := make([]string, len(modelIds))
	args := make([]interface{}, len(modelIds)+1)
	args[0] = isFree

	for i, id := range modelIds {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf("UPDATE models SET is_free = ?, updated_at = CURRENT_TIMESTAMP WHERE id IN (%s)",
		strings.Join(placeholders, ","))

	result, err := tx.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// UpdateModelGiftableStatusWithTx 使用事务更新模型赠费状态
func UpdateModelGiftableStatusWithTx(tx *sql.Tx, modelIds []string, isGiftable bool) (int, error) {
	if tx == nil {
		return 0, fmt.Errorf("事务对象为空")
	}

	if len(modelIds) == 0 {
		return 0, nil
	}

	// 构建IN查询参数
	placeholders := make([]string, len(modelIds))
	args := make([]interface{}, len(modelIds)+1)
	args[0] = isGiftable

	for i, id := range modelIds {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf("UPDATE models SET is_giftable = ?, updated_at = CURRENT_TIMESTAMP WHERE id IN (%s)",
		strings.Join(placeholders, ","))

	result, err := tx.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}
