#!/bin/bash
# 使用 shizhengpt 模型校验中医文档

API_URL="http://192.168.50.17:1234/v1/chat/completions"
MODEL="shizhengpt-7b-vl-i1"

# 读取文档内容
DOC_FILE="$1"
DOC_CONTENT=$(cat "$DOC_FILE")

# 创建校验提示
PROMPT="请作为中医专家，对以下《伤寒论》方剂文档进行校验，检查：
1. 原文引用是否准确
2. 方剂组成是否正确
3. 方证要点是否完整
4. 药证对应是否合理
5. 是否有需要补充或修正的内容

请以JSON格式输出校验结果，包含：
- accuracy: 准确性评分(1-10)
- issues: 发现的问题列表
- suggestions: 改进建议列表

文档内容：
$DOC_CONTENT"

# 调用API
curl -s "$API_URL" \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL\",
    \"messages\": [
      {\"role\": \"system\", \"content\": \"你是中医专家，擅长《伤寒论》方剂学。请对文档进行专业校验。\"},
      {\"role\": \"user\", \"content\": \"$PROMPT\"}
    ],
    \"temperature\": 0.3,
    \"max_tokens\": 2000
  }" | jq -r '.choices[0].message.content'