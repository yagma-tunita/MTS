// Package notify 提供邮件和短信通知发送功能。
//
// 包含的三个组件：
//   - Provider:  通知提供商聚合器，同时持有 EmailSender 和 SMSProvider。
//   - EmailSender: 邮件发送器，支持 STARTTLS（587）和 SSL（465）两种 SMTP 方式。
//   - SMSProvider: 短信发送器接口，有三种实现：
//     - aliyunSMS:   阿里云短信 API（dysmsapi.aliyuncs.com，HMAC-SHA1 签名）
//     - tencentSMS:  腾讯云短信 API（sms.tencentcloudapi.com，TC3-HMAC-SHA256 签名）
//     - consoleSMS:  控制台输出（开发/测试用，只打 log 不实际发送）
//
// 这些组件在 main 函数中根据 config 的 Notify 段配置进行初始化，
// 然后注入到 service.NotificationService 中。当发送通知时，
// 如果通知的 data 中包含 email 或 phone 字段，会触发对应的发送。
package notify

// Provider 聚合邮件和短信通知提供商。
//
// 字段说明：
//   - Email: 邮件发送器。通过 SMTP 协议发送邮件。
//     支持纯文本 (Send) 和 HTML (SendHTML) 两种格式。
//     IsConfigured() 检查是否配置了 SMTP 地址和发件人。
//     如果未配置，NotificationService 调用 Send 时会跳过邮件发送。
//   - SMS: 短信发送器接口。根据配置的 Provider 字段决定具体实现：
//     "aliyun" → 阿里云短信 / "tencent" → 腾讯云短信 / 其他 → 仅打印日志。
//     如果未配置（Provider 为空），默认使用 consoleSMS（仅日志）。
type Provider struct {
	Email *EmailSender // 邮件发送器（nil 安全——NewEmailSender 总是返回非空实例）
	SMS   SMSProvider  // 短信发送器（由 NewSMSProvider 根据配置创建对应实现）
}

// NewProvider 根据配置创建通知提供商。
//
// 无论配置是否完整，NewProvider 总是返回非空实例。
// Email 和 SMS 的可用性由各自的 IsConfigured 或 Provider 类型决定。
// 这样 NotificationService 不需要做空指针检查。
func NewProvider(emailCfg EmailConfig, smsCfg SMSConfig) *Provider {
	return &Provider{
		Email: NewEmailSender(emailCfg),
		SMS:   NewSMSProvider(smsCfg),
	}
}
