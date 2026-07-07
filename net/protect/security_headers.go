package protect

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 返回一个设置安全 HTTP 响应头的中间件。
//
// 设置的响应头及其作用：
//
//  1. X-Content-Type-Options: nosniff
//     禁止浏览器自动推测 Content-Type（MIME 嗅探）。
//     如果服务器返回 Content-Type: text/plain，但浏览器发现内容
//     像 HTML，会将其当作 HTML 渲染。nosniff 禁止这种行为，
//     防止上传文件中的 XSS 攻击。
//
//  2. X-Frame-Options: DENY
//     禁止页面被嵌入到 iframe 中，防止点击劫持（Clickjacking）。
//     DENY 是最严格的值，任何页面都不能将此页面嵌入 iframe。
//     如果将来需要允许特定域名嵌入，可改为 ALLOW-FROM uri。
//
//  3. X-XSS-Protection: 1; mode=block
//     启用浏览器的 XSS 过滤功能。当检测到反射型 XSS 攻击时，
//     浏览器会阻止页面渲染而不是尝试过滤（过滤往往不够安全）。
//     注意：现代浏览器已经逐步废弃这个头，但设置它不会有副作用。
//
//  4. Referrer-Policy: strict-origin-when-cross-origin
//     控制 Referer 头中发送的信息量。
//     同源请求：发送完整的 URL。
//     跨源请求：只发送 origin（协议+域名+端口），不发送路径和查询参数。
//     从 HTTPS 到 HTTP：不发送 Referer。
//     这防止了 URL 中的敏感参数（如 token）泄露给第三方站点。
//
//  5. Content-Security-Policy: default-src 'self'
//     内容安全策略，只允许加载同源资源。
//     含义：
//     - 脚本（script-src）：只允许同源脚本
//     - 样式（style-src）：只允许同源样式表
//     - 图片（img-src）：只允许同源图片
//     - 字体（font-src）：只允许同源字体
//     - 等...
//     这可以防御 XSS 攻击——即使攻击者注入了恶意脚本标签，
//     浏览器也不会执行非同源的脚本。
//     注意：如果前端使用了 CDN 资源（如 bootstrap 的 CSS），
//     需要修改此策略添加 CDN 域名白名单。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}
