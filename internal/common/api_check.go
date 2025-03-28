/**
  @author: Hanhai
  @since: 2025/3/17 11:55:04
  @desc: API 测试函数
**/

package common

import (
	"bytes"
	"encoding/json"
	"flowsilicon/internal/config"
	"flowsilicon/internal/logger"
	"flowsilicon/pkg/utils"
	"fmt"
	"io"
	"net/http"
)

// TestChatAPI 测试对话API是否正常工作
func TestChatAPI(apiKey string) (bool, string, error) {

	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL
	targetURL := fmt.Sprintf("%s/v1/chat/completions", baseURL)

	logger.Info("测试对话API, 目标URL: %s", targetURL)

	// 构建请求体
	requestBody := map[string]interface{}{
		"model": "Qwen/Qwen2.5-7B-Instruct",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "你好，这是一个测试",
			},
		},
		"max_tokens": 512,
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("序列化请求体失败: %v\n", err)
		return false, "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	fmt.Printf("对话测试请求体: %s\n", string(jsonBody))

	// 创建请求
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return false, "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept-Encoding", "identity")

	// 创建HTTP客户端
	client := utils.CreateClient()

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return false, "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体失败: %v\n", err)
		return false, "", fmt.Errorf("读取响应体失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("请求失败，状态码: %d, 响应: %s\n", resp.StatusCode, string(respBody))
		return false, string(respBody), fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 尝试解析响应体
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		fmt.Printf("解析响应体失败: %v\n", err)
		return false, string(respBody), fmt.Errorf("解析响应体失败: %v", err)
	}

	// 检查是否包含choices字段
	if choices, ok := responseData["choices"]; ok {
		fmt.Printf("检测到choices字段,类型: %T\n", choices)
		if choicesArray, ok := choices.([]interface{}); ok && len(choicesArray) > 0 {
			fmt.Printf("choices字段是数组,长度: %d\n", len(choicesArray))
			return true, string(respBody), nil
		}
	}

	return false, string(respBody), fmt.Errorf("响应中未找到有效的choices字段")
}

// testImageGeneration 测试图片生成API是否正常工作
func TestImageGeneration(apiKey string) (bool, string, error) {
	// 构建请求URL
	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL
	targetURL := fmt.Sprintf("%s/v1/images/generations", baseURL)

	logger.Info("测试图片生成API，目标URL: %s", targetURL)

	// 构建请求体
	requestBody := map[string]interface{}{
		"model":               "Kwai-Kolors/Kolors",
		"prompt":              "a beautiful landscape with mountains and lake",
		"image_size":          "512x512", // 使用较小的尺寸加快测试
		"batch_size":          1,
		"num_inference_steps": 10, // 减少步数加快测试
		"guidance_scale":      7.5,
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("序列化请求体失败: %v", err)
		return false, "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	logger.Info("图片生成测试请求体: %s", string(jsonBody))

	// 创建请求
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error("创建请求失败: %v", err)
		return false, "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept-Encoding", "identity")

	logger.Info("图片生成测试使用的API密钥: %s", utils.MaskKey(apiKey))

	// 创建HTTP客户端
	client := utils.CreateClient()

	// 发送请求
	logger.Info("正在发送图片生成测试请求...")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送请求失败: %v", err)
		return false, "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("图片生成测试请求状态码: %d", resp.StatusCode)

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应体失败: %v", err)
		return false, "", fmt.Errorf("读取响应体失败: %v", err)
	}

	logger.Info("图片生成测试响应体: %s", string(respBody))

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return false, string(respBody), fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 尝试解析响应体
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		logger.Error("解析响应体失败: %v", err)
		return false, string(respBody), fmt.Errorf("解析响应体失败: %v", err)
	}

	// 检查是否包含images字段
	success := false
	if images, ok := responseData["images"]; ok {
		logger.Info("检测到images字段，类型: %T", images)
		if imagesArray, ok := images.([]interface{}); ok && len(imagesArray) > 0 {
			// 检查第一个元素是否包含url字段
			if imgObj, ok := imagesArray[0].(map[string]interface{}); ok {
				if _, hasUrl := imgObj["url"]; hasUrl {
					success = true
					logger.Info("图片生成API测试成功，找到有效的图片URL")
				}
			} else if _, ok := imagesArray[0].(string); ok {
				// 如果是字符串，也认为是成功的
				success = true
				logger.Info("图片生成API测试成功，找到有效的图片URL字符串")
			}
		}
	}

	// 记录所有响应字段，便于调试
	keys := make([]string, 0, len(responseData))
	for k := range responseData {
		keys = append(keys, k)
	}
	logger.Info("响应中的所有字段: %v", keys)

	if !success {
		return false, string(respBody), fmt.Errorf("响应中未找到有效的images字段")
	}

	return true, string(respBody), nil
}

// testModelsAPI 测试模型列表API是否正常工作
func TestModelsAPI(apiKey string) (bool, string, error) {
	// 构建请求URL
	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL
	targetURL := fmt.Sprintf("%s/v1/models", baseURL)

	logger.Info("测试模型列表API，目标URL: %s", targetURL)

	// 创建请求
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		logger.Error("创建请求失败: %v", err)
		return false, "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept-Encoding", "identity")

	logger.Info("模型列表测试使用的API密钥: %s", utils.MaskKey(apiKey))

	// 创建HTTP客户端
	client := utils.CreateClient()

	// 发送请求
	logger.Info("正在发送模型列表测试请求...")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送请求失败: %v", err)
		return false, "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("模型列表测试请求状态码: %d", resp.StatusCode)

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应体失败: %v", err)
		return false, "", fmt.Errorf("读取响应体失败: %v", err)
	}

	logger.Info("模型列表测试响应体: %s", string(respBody))

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return false, string(respBody), fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 尝试解析响应体
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		logger.Error("解析响应体失败: %v", err)
		return false, string(respBody), fmt.Errorf("解析响应体失败: %v", err)
	}

	// 检查是否包含data字段
	if data, ok := responseData["data"]; ok {
		logger.Info("检测到data字段，类型: %T", data)
		if dataArray, ok := data.([]interface{}); ok && len(dataArray) > 0 {
			logger.Info("data字段是数组，长度: %d", len(dataArray))
			return true, string(respBody), nil
		}
	}

	// 记录所有响应字段，便于调试
	keys := make([]string, 0, len(responseData))
	for k := range responseData {
		keys = append(keys, k)
	}
	logger.Info("响应中的所有字段: %v", keys)

	return false, string(respBody), fmt.Errorf("响应中未找到有效的data字段")
}

// testRerankAPI 测试重排序API是否正常工作
func TestRerankAPI(apiKey string) (bool, string, error) {
	// 构建请求URL
	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL
	targetURL := fmt.Sprintf("%s/v1/rerank", baseURL)

	logger.Info("测试重排序API，目标URL: %s", targetURL)

	// 构建请求体
	requestBody := map[string]interface{}{
		"model": "BAAI/bge-reranker-v2-m3",
		"query": "Apple",
		"documents": []string{
			"apple",
			"banana",
			"fruit",
			"vegetable",
		},
		"top_n":              4,
		"return_documents":   false,
		"max_chunks_per_doc": 1024,
		"overlap_tokens":     80,
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("序列化请求体失败: %v", err)
		return false, "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	logger.Info("重排序测试请求体: %s", string(jsonBody))

	// 创建请求
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error("创建请求失败: %v", err)
		return false, "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept-Encoding", "identity")

	logger.Info("重排序测试使用的API密钥: %s", utils.MaskKey(apiKey))

	// 创建HTTP客户端
	client := utils.CreateClient()

	// 发送请求
	logger.Info("正在发送重排序测试请求...")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送请求失败: %v", err)
		return false, "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	logger.Info("重排序测试请求状态码: %d", resp.StatusCode)

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应体失败: %v", err)
		return false, "", fmt.Errorf("读取响应体失败: %v", err)
	}

	logger.Info("重排序测试响应体: %s", string(respBody))

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return false, string(respBody), fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	// 尝试解析响应体
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		logger.Error("解析响应体失败: %v", err)
		return false, string(respBody), fmt.Errorf("解析响应体失败: %v", err)
	}

	// 检查是否包含results字段
	if results, ok := responseData["results"]; ok {
		logger.Info("检测到results字段，类型: %T", results)
		if resultsArray, ok := results.([]interface{}); ok && len(resultsArray) > 0 {
			logger.Info("results字段是数组，长度: %d", len(resultsArray))
			return true, string(respBody), nil
		}
	}

	// 记录所有响应字段，便于调试
	keys := make([]string, 0, len(responseData))
	for k := range responseData {
		keys = append(keys, k)
	}
	logger.Info("响应中的所有字段: %v", keys)

	return false, string(respBody), fmt.Errorf("响应中未找到有效的results字段")
}

// TestEmbeddings 测试embeddings API是否正常工作
func TestEmbeddings(apiKey string) (bool, string, error) {
	// 构建请求URL
	cfg := config.GetConfig()
	baseURL := cfg.ApiProxy.BaseURL
	targetURL := fmt.Sprintf("%s/v1/embeddings", baseURL)

	logger.Info("测试embeddings API, 目标URL: %s", targetURL)

	// 构建请求体
	requestBody := map[string]interface{}{
		"model": "BAAI/bge-m3",
		"input": []string{"测试嵌入功能"},
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("序列化请求体失败: %v", err)
		return false, "", fmt.Errorf("序列化请求体失败: %v", err)
	}

	logger.Info("embeddings测试请求体: %s", string(jsonBody))

	// 创建请求
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error("创建请求失败: %v", err)
		return false, "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept-Encoding", "identity")

	// 创建HTTP客户端
	client := utils.CreateClient()

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("发送请求失败: %v", err)
		return false, "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应体失败: %v", err)
		return false, "", fmt.Errorf("读取响应体失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Error("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
		return false, string(respBody), fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var response map[string]interface{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		logger.Error("解析响应失败: %v", err)
		return false, string(respBody), fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应是否包含embeddings
	if _, ok := response["data"]; !ok {
		logger.Error("响应中没有embeddings数据: %s", string(respBody))
		return false, string(respBody), fmt.Errorf("响应中没有embeddings数据")
	}

	logger.Info("embeddings测试成功")
	return true, string(respBody), nil
}
