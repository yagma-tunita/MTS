package notify

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SMSProvider 短信发送器接口。
type SMSProvider interface {
	Send(phone, message string) error
}

// SMSConfig 短信提供商配置。
type SMSConfig struct {
	Provider        string // 可选值："aliyun", "tencent", "console"
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
}

// NewSMSProvider 根据配置创建短信发送器。
// 未指定或未知提供商时使用 consoleSMS（仅输出日志）。
func NewSMSProvider(cfg SMSConfig) SMSProvider {
	switch cfg.Provider {
	case "aliyun":
		return &aliyunSMS{cfg: cfg}
	case "tencent":
		return &tencentSMS{cfg: cfg}
	default:
		return &consoleSMS{}
	}
}

// ══════ Console（开发/测试环境）══════

type consoleSMS struct{}

// Send 将短信内容输出到日志，不实际发送。
func (c *consoleSMS) Send(phone, message string) error {
	slog.Info("[SMS] console send", "phone", phone, "message", message)
	return nil
}

// ══════ 阿里云短信 ══════

type aliyunSMS struct {
	cfg SMSConfig
}

// Send 调用阿里云短信 API（dysmsapi.aliyuncs.com）发送短信。
// 使用 HMAC-SHA1 签名，POST 方式请求。
func (a *aliyunSMS) Send(phone, message string) error {
	params := map[string]string{
		"Action":           "SendSms",
		"Version":          "2017-05-25",
		"RegionId":         "cn-hangzhou",
		"PhoneNumbers":     phone,
		"SignName":         a.cfg.SignName,
		"TemplateCode":     a.cfg.TemplateCode,
		"TemplateParam":    fmt.Sprintf(`{"code":"%s"}`, message),
		"Format":           "JSON",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.New().String(),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"AccessKeyId":      a.cfg.AccessKeyID,
	}

	// 参数排序并构造待签名字符串
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var qParts []string
	for _, k := range keys {
		qParts = append(qParts, percentEncode(k)+"="+percentEncode(params[k]))
	}
	queryStr := strings.Join(qParts, "&")

	stringToSign := "POST&" + percentEncode("/") + "&" + percentEncode(queryStr)

	mac := hmac.New(sha1.New, []byte(a.cfg.AccessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params["Signature"] = signature
	qParts2 := make([]string, 0, len(params))
	keys2 := make([]string, 0, len(params))
	for k := range params {
		keys2 = append(keys2, k)
	}
	sort.Strings(keys2)
	for _, k := range keys2 {
		qParts2 = append(qParts2, percentEncode(k)+"="+percentEncode(params[k]))
	}

	endpoint := "https://dysmsapi.aliyuncs.com/?" + strings.Join(qParts2, "&")

	resp, err := http.Post(endpoint, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("aliyun sms request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("aliyun sms decode: %w", err)
	}
	if result.Code != "OK" {
		return fmt.Errorf("aliyun sms failed: %s - %s", result.Code, result.Message)
	}
	slog.Info("[AliyunSMS] sent", "phone", phone, "template", a.cfg.TemplateCode)
	return nil
}

// percentEncode 按照阿里云签名规范进行 URL 编码。
func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// ══════ 腾讯云短信 ══════

type tencentSMS struct {
	cfg SMSConfig
}

// Send 调用腾讯云短信 API（sms.tencentcloudapi.com）发送短信。
// 使用 TC3-HMAC-SHA256 签名认证。
func (t *tencentSMS) Send(phone, message string) error {
	service := "sms"
	host := "sms.tencentcloudapi.com"
	region := "ap-guangzhou"
	action := "SendSms"
	version := "2021-01-11"
	algorithm := "TC3-HMAC-SHA256"
	timestamp := time.Now().Unix()
	dateStr := time.Now().UTC().Format("2006-01-02")

	payload := map[string]interface{}{
		"PhoneNumberSet":   []string{phone},
		"TemplateID":       t.cfg.TemplateCode,
		"TemplateParamSet": []string{message},
		"SmsSdkAppId":      t.cfg.AccessKeyID,
		"SignName":         t.cfg.SignName,
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadHash := sha256Hex(payloadBytes)

	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\n", host)
	signedHeaders := "content-type;host"

	canonicalRequest := fmt.Sprintf("POST\n/\n\n%s\n%s\n%s",
		canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/tc3_request", dateStr, service)
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm, timestamp, credentialScope, sha256Hex([]byte(canonicalRequest)))

	secretDate := hmacSHA256([]byte("TC3"+t.cfg.AccessKeySecret), dateStr)
	secretService := hmacSHA256(secretDate, service)
	signingKey := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, t.cfg.AccessKeyID, credentialScope, signedHeaders, signature)

	req, _ := http.NewRequest("POST", "https://"+host, strings.NewReader(string(payloadBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-TC-Region", region)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("tencent sms request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Response struct {
			SendStatusSet []struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"SendStatusSet"`
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error,omitempty"`
			RequestId string `json:"RequestId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("tencent sms decode: %w", err)
	}
	if result.Response.Error != nil {
		return fmt.Errorf("tencent sms failed: %s - %s",
			result.Response.Error.Code, result.Response.Error.Message)
	}
	if len(result.Response.SendStatusSet) > 0 && result.Response.SendStatusSet[0].Code != "Ok" {
		return fmt.Errorf("tencent sms failed: %s - %s",
			result.Response.SendStatusSet[0].Code, result.Response.SendStatusSet[0].Message)
	}

	slog.Info("[TencentSMS] sent", "phone", phone, "template", t.cfg.TemplateCode)
	return nil
}

// sha256Hex 计算 SHA-256 哈希并返回十六进制字符串。
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hmacSHA256 使用 HMAC-SHA256 算法计算消息认证码。
func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

