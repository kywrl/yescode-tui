# YesCode API 文档

## 📋 目录
- [基本信息](#基本信息)
- [认证](#认证)
- [错误处理](#错误处理)
- [API 端点](#api-端点)
  - [用户余额](#1-获取用户余额)
  - [提供商列表](#2-获取可用提供商列表)
  - [提供商替代方案](#3-获取提供商替代方案)
  - [当前提供商选择](#4-获取当前提供商选择)
  - [切换提供商](#5-切换提供商)
  - [余额使用偏好](#6-设置余额使用偏好)
  - [用户资料](#7-获取用户资料)

---

## 基本信息

### Base URL
```
https://co.yes.vg
```

### 请求格式
- **Content-Type**: `application/json`
- **Accept**: `application/json`

### 超时设置
- **连接超时**: 10 秒
- **最大超时**: 30 秒

---

## 认证

所有 API 请求需要在 HTTP Header 中包含 API Key：

```http
X-API-Key: your-api-key-here
```

### 认证失败
当 API Key 无效或过期时，返回 `401 Unauthorized`：

```json
{
  "error": "Unauthorized",
  "message": "Invalid API Key"
}
```

---

## 错误处理

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 401 | 认证失败（API Key 无效） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 错误响应格式

```json
{
  "error": "错误类型",
  "message": "错误详细信息"
}
```

---

## API 端点

### 1. 获取用户余额

获取当前用户的账户余额信息，包括订阅余额、按需付费余额、本周消费等。

#### 请求

```http
GET /api/v1/user/balance
```

**Headers:**
```http
X-API-Key: {your-api-key}
Accept: application/json
```

#### 响应示例

```json
{
  "subscription_balance": 45.23,
  "pay_as_you_go_balance": 12.00,
  "total_balance": 57.23,
  "credit_balance": 0.00,
  "weekly_limit": 15.00,
  "weekly_spent_balance": 8.50
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `subscription_balance` | number | 订阅套餐余额（美元） |
| `pay_as_you_go_balance` | number | 按需付费余额（美元） |
| `total_balance` | number | 总余额（美元） |
| `credit_balance` | number | 信用额度（美元） |
| `weekly_limit` | number | 本周消费限额（美元） |
| `weekly_spent_balance` | number | 本周已消费金额（美元） |

#### 使用示例

```bash
curl -X GET "https://co.yes.vg/api/v1/user/balance" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: application/json"
```

---

### 2. 获取可用提供商列表

获取当前用户可用的提供商分组列表。

#### 请求

```http
GET /api/v1/user/available-providers
```

**Headers:**
```http
X-API-Key: {your-api-key}
Accept: application/json
```

#### 响应示例

```json
{
  "has_payg_balance": true,
  "has_subscription": true,
  "providers": [
    {
      "provider": {
        "id": 3,
        "display_name": "GPT-4 Turbo",
        "type": "openai",
        "description": "OpenAI GPT-4 Turbo model"
      },
      "rate_multiplier": 1.0,
      "is_default": true,
      "source": "subscription"
    },
    {
      "provider": {
        "id": 5,
        "display_name": "Claude Sonnet",
        "type": "anthropic",
        "description": "Anthropic Claude 3.5 Sonnet"
      },
      "rate_multiplier": 1.2,
      "is_default": false,
      "source": "pay_as_you_go"
    }
  ]
}
```

#### 响应字段说明

**顶层字段:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `has_payg_balance` | boolean | 是否有按需付费余额 |
| `has_subscription` | boolean | 是否有订阅套餐 |
| `providers` | array | 提供商列表 |

**providers 数组项:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `provider.id` | integer | 提供商 ID |
| `provider.display_name` | string | 显示名称 |
| `provider.type` | string | 提供商类型 |
| `provider.description` | string | 描述信息 |
| `rate_multiplier` | number | 费率倍数（相对于基准价格） |
| `is_default` | boolean | 是否为默认提供商 |
| `source` | string | 来源：`subscription` 或 `pay_as_you_go` |

#### 使用示例

```bash
curl -X GET "https://co.yes.vg/api/v1/user/available-providers" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: application/json"
```

---

### 3. 获取提供商替代方案

获取指定提供商分组内的所有可用替代方案。

#### 请求

```http
GET /api/v1/user/provider-alternatives/{provider_id}
```

**路径参数:**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `provider_id` | integer | 是 | 提供商分组 ID |

**Headers:**
```http
X-API-Key: {your-api-key}
Accept: application/json
```

#### 响应示例

```json
{
  "data": [
    {
      "is_self": true,
      "alternative": {
        "id": 101,
        "display_name": "OpenAI (Official)",
        "type": "openai",
        "rate_multiplier": 1.0,
        "description": "Official OpenAI API"
      }
    },
    {
      "is_self": false,
      "alternative": {
        "id": 102,
        "display_name": "Cloudflare Workers AI",
        "type": "openai",
        "rate_multiplier": 1.2,
        "description": "Cloudflare proxy"
      }
    },
    {
      "is_self": false,
      "alternative": {
        "id": 103,
        "display_name": "Azure OpenAI",
        "type": "openai",
        "rate_multiplier": 0.9,
        "description": "Azure hosted"
      }
    }
  ]
}
```

#### 响应字段说明

**data 数组项:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `is_self` | boolean | 是否为原始/官方方案 |
| `alternative.id` | integer | 替代方案 ID |
| `alternative.display_name` | string | 显示名称 |
| `alternative.type` | string | 类型 |
| `alternative.rate_multiplier` | number | 费率倍数 |
| `alternative.description` | string | 描述 |

#### 使用示例

```bash
curl -X GET "https://co.yes.vg/api/v1/user/provider-alternatives/3" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: application/json"
```

---

### 4. 获取当前提供商选择

获取指定提供商分组当前生效的替代方案。

#### 请求

```http
GET /api/v1/user/provider-alternatives/{provider_id}/selection
```

**路径参数:**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `provider_id` | integer | 是 | 提供商分组 ID |

**Headers:**
```http
X-API-Key: {your-api-key}
Accept: application/json
```

#### 响应示例

```json
{
  "data": {
    "provider_id": 3,
    "selected_alternative_id": 101,
    "selected_alternative": {
      "id": 101,
      "display_name": "OpenAI (Official)",
      "type": "openai",
      "rate_multiplier": 1.0,
      "description": "Official OpenAI API"
    }
  }
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `data.provider_id` | integer | 提供商分组 ID |
| `data.selected_alternative_id` | integer | 当前选中的替代方案 ID |
| `data.selected_alternative` | object | 选中方案的详细信息 |

#### 使用示例

```bash
curl -X GET "https://co.yes.vg/api/v1/user/provider-alternatives/3/selection" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: application/json"
```

---

### 5. 切换提供商

切换指定提供商分组的生效替代方案。

#### 请求

```http
PUT /api/v1/user/provider-alternatives/{provider_id}/selection
```

**路径参数:**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `provider_id` | integer | 是 | 提供商分组 ID |

**Headers:**
```http
X-API-Key: {your-api-key}
Content-Type: application/json
Accept: application/json
```

**请求体:**

```json
{
  "selected_alternative_id": 102
}
```

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `selected_alternative_id` | integer | 是 | 要切换到的替代方案 ID |

#### 响应示例

```json
{
  "data": {
    "provider_id": 3,
    "selected_alternative_id": 102,
    "selected_alternative": {
      "id": 102,
      "display_name": "Cloudflare Workers AI",
      "type": "openai",
      "rate_multiplier": 1.2,
      "description": "Cloudflare proxy"
    }
  }
}
```

#### 响应字段说明

与 [获取当前提供商选择](#4-获取当前提供商选择) 相同。

#### 使用示例

```bash
curl -X PUT "https://co.yes.vg/api/v1/user/provider-alternatives/3/selection" \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"selected_alternative_id": 102}'
```

---

### 6. 设置余额使用偏好

设置账户余额的使用偏好（优先订阅或仅按需付费）。

#### 请求

```http
PUT /api/v1/user/balance-preference
```

**Headers:**
```http
X-API-Key: {your-api-key}
Content-Type: application/json
Accept: application/json
```

**请求体:**

```json
{
  "balance_preference": "subscription_first"
}
```

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `balance_preference` | string | 是 | 余额偏好：`subscription_first` 或 `payg_only` |

**balance_preference 可选值:**

| 值 | 说明 |
|----|------|
| `subscription_first` | 优先使用订阅余额，订阅用完后使用按需付费 |
| `payg_only` | 仅使用按需付费余额（无 OPUS 使用限制） |

#### 响应示例

```json
{
  "balance_preference": "subscription_first",
  "updated_at": "2025-11-13T14:23:56Z"
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `balance_preference` | string | 当前的余额偏好 |
| `updated_at` | string | 更新时间（ISO 8601 格式） |

#### 使用示例

```bash
curl -X PUT "https://co.yes.vg/api/v1/user/balance-preference" \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"balance_preference": "payg_only"}'
```

---

### 7. 获取用户资料

获取当前用户的完整资料信息，包括用户信息、余额、订阅套餐等。

#### 请求

```http
GET /api/v1/auth/profile
```

**Headers:**
```http
X-API-Key: {your-api-key}
Accept: application/json
```

#### 响应示例

```json
{
  "email": "user@example.com",
  "username": "张三",
  "balance": 57.23,
  "subscription_balance": 45.23,
  "pay_as_you_go_balance": 12.00,
  "balance_preference": "subscription_first",
  "subscription_expiry": "2025-12-31T23:59:59Z",
  "current_week_spend": 8.50,
  "current_month_spend": 32.15,
  "subscription_plan": {
    "name": "Pro Plan",
    "price": 49.99,
    "is_active": true,
    "daily_balance": 5.00,
    "weekly_limit": 15.00,
    "monthly_spend_limit": 100.00
  }
}
```

#### 响应字段说明

**用户信息:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `email` | string | 用户邮箱 |
| `username` | string | 用户名 |

**余额信息:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `balance` | number | 总余额（美元） |
| `subscription_balance` | number | 订阅余额（美元） |
| `pay_as_you_go_balance` | number | 按需付费余额（美元） |
| `balance_preference` | string | 余额使用偏好 |
| `subscription_expiry` | string | 订阅到期时间（ISO 8601） |

**消费信息:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `current_week_spend` | number | 本周消费（美元） |
| `current_month_spend` | number | 本月消费（美元） |

**订阅套餐:**

| 字段 | 类型 | 说明 |
|------|------|------|
| `subscription_plan.name` | string | 套餐名称 |
| `subscription_plan.price` | number | 套餐价格（美元/月） |
| `subscription_plan.is_active` | boolean | 是否有效 |
| `subscription_plan.daily_balance` | number | 日消费限额（美元） |
| `subscription_plan.weekly_limit` | number | 周消费限额（美元） |
| `subscription_plan.monthly_spend_limit` | number | 月消费限额（美元） |

#### 使用示例

```bash
curl -X GET "https://co.yes.vg/api/v1/auth/profile" \
  -H "X-API-Key: your-api-key" \
  -H "Accept: application/json"
```

---

## 术语说明

### 分组 (Provider Group)
- API 端点中的 `provider_id` 参数指的是**提供商分组**
- 一个分组代表一组相关的 AI 服务提供商（如 GPT-4 系列）
- 分组内可以包含多个替代方案（如官方 API、Cloudflare 代理、Azure 等）

### 替代方案 (Alternative)
- 同一分组内的不同实现方式或服务源
- 每个替代方案有自己的费率倍数 (`rate_multiplier`)
- 用户可以在分组内自由切换替代方案

### 费率倍数 (Rate Multiplier)
- 相对于基准价格的倍数
- `1.0` = 标准价格
- `1.2` = 比标准价格贵 20%
- `0.9` = 比标准价格便宜 10%

---

## 代码示例

### Bash 示例（使用 yc 脚本中的实现）

```bash
# 设置 API Key 和 Base URL
API_KEY="your-api-key"
API_BASE_URL="https://co.yes.vg"

# 获取余额
curl -s -w "\n%{http_code}" \
  -X GET \
  -H "accept: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  --connect-timeout 10 \
  --max-time 30 \
  "${API_BASE_URL}/api/v1/user/balance"

# 获取提供商列表
curl -s -w "\n%{http_code}" \
  -X GET \
  -H "accept: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  "${API_BASE_URL}/api/v1/user/available-providers"

# 切换提供商
curl -s -w "\n%{http_code}" \
  -X PUT \
  -H "accept: application/json" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${API_KEY}" \
  -d '{"selected_alternative_id": 102}' \
  "${API_BASE_URL}/api/v1/user/provider-alternatives/3/selection"
```

### Python 示例

```python
import requests

API_KEY = "your-api-key"
BASE_URL = "https://co.yes.vg"

headers = {
    "X-API-Key": API_KEY,
    "Accept": "application/json"
}

# 获取余额
response = requests.get(f"{BASE_URL}/api/v1/user/balance", headers=headers)
balance = response.json()
print(f"总余额: ${balance['total_balance']}")

# 获取提供商列表
response = requests.get(f"{BASE_URL}/api/v1/user/available-providers", headers=headers)
providers = response.json()
for p in providers['providers']:
    print(f"{p['provider']['display_name']} - 费率: ×{p['rate_multiplier']}")

# 切换提供商
headers["Content-Type"] = "application/json"
data = {"selected_alternative_id": 102}
response = requests.put(
    f"{BASE_URL}/api/v1/user/provider-alternatives/3/selection",
    headers=headers,
    json=data
)
result = response.json()
print(f"已切换到: {result['data']['selected_alternative']['display_name']}")
```

### JavaScript 示例

```javascript
const API_KEY = 'your-api-key';
const BASE_URL = 'https://co.yes.vg';

const headers = {
  'X-API-Key': API_KEY,
  'Accept': 'application/json'
};

// 获取余额
async function getBalance() {
  const response = await fetch(`${BASE_URL}/api/v1/user/balance`, { headers });
  const balance = await response.json();
  console.log(`总余额: $${balance.total_balance}`);
}

// 获取提供商列表
async function getProviders() {
  const response = await fetch(`${BASE_URL}/api/v1/user/available-providers`, { headers });
  const data = await response.json();
  data.providers.forEach(p => {
    console.log(`${p.provider.display_name} - 费率: ×${p.rate_multiplier}`);
  });
}

// 切换提供商
async function switchProvider(providerId, alternativeId) {
  const response = await fetch(
    `${BASE_URL}/api/v1/user/provider-alternatives/${providerId}/selection`,
    {
      method: 'PUT',
      headers: {
        ...headers,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ selected_alternative_id: alternativeId })
    }
  );
  const result = await response.json();
  console.log(`已切换到: ${result.data.selected_alternative.display_name}`);
}

// 执行
getBalance();
getProviders();
switchProvider(3, 102);
```

---

## 最佳实践

### 1. 错误处理

```bash
# 始终检查 HTTP 状态码
response=$(curl -s -w "\n%{http_code}" ...)
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n -1)

if [[ "$http_code" == "401" ]]; then
    echo "认证失败，请检查 API Key"
    exit 1
elif [[ "$http_code" != "200" ]]; then
    echo "请求失败: HTTP $http_code"
    echo "响应: $body"
    exit 1
fi
```

### 2. 超时设置

```bash
# 设置合理的超时时间
curl --connect-timeout 10 --max-time 30 ...
```

### 3. 重试机制

```bash
# 对于 401 错误，提示用户重新输入 API Key
if [[ "$http_code" == "401" ]]; then
    read -p "请重新输入 API Key: " API_KEY
    # 重试请求
fi
```

### 4. API Key 安全

```bash
# 不要在代码中硬编码 API Key
# 使用配置文件
API_KEY=$(jq -r '.api_key' ~/.yescode/config.json)

# 设置正确的文件权限
chmod 600 ~/.yescode/config.json
```

### 5. 并行请求

```bash
# 当需要获取多个数据时，使用并行请求
(api_get_balance > temp_balance.json) &
pid_balance=$!
(api_get_providers > temp_providers.json) &
pid_providers=$!

wait $pid_balance
wait $pid_providers

# 处理结果
```

---

## 更新日志

### 2025-11-13
- 初始版本
- 记录所有已知的 API 端点
- 添加详细的请求/响应示例
- 添加多语言代码示例

---

## 反馈与支持

如有 API 相关问题或建议，请联系 YesCode 支持团队。

**API 状态监控**: https://status.yes.vg (如有)

---

_文档生成时间: 2025-11-13_
