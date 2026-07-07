package middleware

import (
	"sync"

	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig 浠ょ墝妗堕檺娴佸櫒閰嶇疆銆?
//
// 浠ょ墝妗剁畻娉曪細
//   - 妗跺閲忥紙Burst锛夛細鍏佽鐨勭獊鍙戣姹傛暟銆傛《婊℃椂鏈€澶氬彲鎺ュ彈 Burst 涓繛缁姹傘€?
//   - 濉厖閫熺巼锛圧ate锛夛細姣忕鍚戞《涓坊鍔犵殑浠ょ墝鏁般€傚钩鍧囪姹傞€熺巼涓嶈兘瓒呰繃姝ゅ€笺€?
//   - 姣忔璇锋眰娑堣€椾竴涓护鐗屻€傛《绌烘椂璇锋眰琚嫆缁濄€?
//   - 妗朵腑鐨勪护鐗屾暟涓嶄細瓒呰繃 Burst锛堝鍑虹殑婧㈠嚭锛夈€?
//
// 閫傜敤鍦烘櫙锛?
//   - 闃叉鍗曚釜 IP 鐨勬伓鎰忚姹傛墦婊℃湇鍔″櫒璧勬簮銆?
//   - 骞虫粦绐佸彂娴侀噺锛堜护鐗屾《鍏佽涓€瀹氱▼搴︾殑绐佸彂锛屾瘮婕忔《鏇寸伒娲伙級銆?
type RateLimiterConfig struct {
	Rate  rate.Limit // 姣忕浜х敓鐨勪护鐗屾暟锛堝钩鍧囪姹傞€熺巼涓婇檺锛?
	Burst int        // 妗跺閲忥紙鍏佽鐨勭獊鍙戣姹傛暟锛?
}

// DefaultRateLimiterConfig 杩斿洖榛樿闄愭祦閰嶇疆銆?
//
// 榛樿鍊硷細Rate=100, Burst=20銆?
// 鍚箟锛氬钩鍧囨瘡绉掓渶澶?100 涓姹傦紝绐佸彂鏃跺彲杩炵画澶勭悊 20 涓姹傘€?
// 杩欎釜鍊煎熀浜庡亣璁惧崟瀹炰緥宄板€?QPS 绾?1000 鐨?1/10 浣滀负瀹夊叏闃堝€笺€?
// 瀹為檯鐢熶骇涓簲鏍规嵁鍘嬪姏娴嬭瘯缁撴灉璋冩暣銆?
//
// 涓轰粈涔?Rate=100 浣?Burst=20 鑰屼笉鏄?100锛?
//   Burst 璁惧緱灏忎竴浜涘彲浠ラ槻姝㈡伓鎰忕敤鎴峰湪 1 绉掑唴鎵撴弧 100 涓姹?
//   鍚冨厜鏈嶅姟鍣ㄨ祫婧愩€?0 涓獊鍙戣姹傝冻澶熸甯哥敤鎴风殑椤甸潰鍒锋柊鍜屾搷浣溿€?
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Rate:  100,
		Burst: 20,
	}
}

// ipRateLimiter 鍩轰簬 IP 鍦板潃鐨勭嫭绔嬮檺娴佸櫒銆?
//
// 姣忎釜 IP 瀵瑰簲涓€涓嫭绔嬬殑浠ょ墝妗讹紝浜掍笉褰卞搷銆?
// 杩欐牱鏌愪釜 IP 鐨勬伓鎰忚姹備笉浼氬奖鍝嶅叾浠栫敤鎴风殑璁块棶銆?
//
// 瀹夊叏鎬э細
//   - 濡傛灉璇锋眰缁忚繃鍙嶅悜浠ｇ悊锛圢ginx锛夛紝c.ClientIP() 鍙兘杩斿洖
//     浠ｇ悊鐨?IP 鑰岄潪鐪熷疄瀹㈡埛绔?IP銆傞渶瑕佸湪 Gin 涓厤缃?TrustedProxies銆?
//   - 濡傛灉鏀诲嚮鑰呴绻佸垏鎹?IP锛堝浣跨敤浠ｇ悊姹狅級锛屽熀浜?IP 鐨勯檺娴佹晥鏋滄湁闄愩€?
//     杩欑鍦烘櫙闇€瑕佹洿澶嶆潅鐨勯檺娴佺瓥鐣ワ紙濡傚熀浜?Token 鎴?User-Agent锛夈€?
type ipRateLimiter struct {
	ips map[string]*rate.Limiter // 姣忎釜 IP 瀵瑰簲鐨勪护鐗屾《
	mu  sync.RWMutex             // 淇濇姢 ips map 鐨勫苟鍙戝畨鍏?
	cfg RateLimiterConfig
}

// newIPRateLimiter 创建 IP 限流器
func newIPRateLimiter(cfg RateLimiterConfig) *ipRateLimiter {
	return &ipRateLimiter{
		ips: make(map[string]*rate.Limiter),
		cfg: cfg,
	}
}

// getLimiter 获取或创建指定 IP 对应的限流器。
//
// 懒加载模式：只在有新 IP 访问时创建对应的令牌桶。
// 这会在运行期间持续消耗内存（每个 IP 一个结构体）。
// 在正常业务中，活跃 IP 的数量有限（远少于 1 万个），
// 内存占用可以忽略。如果需要严格的资源控制，可以
// 加定时清理不活跃 IP 的逻辑。
func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	limiter, exists := l.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(l.cfg.Rate, l.cfg.Burst)
		l.ips[ip] = limiter
	}
	return limiter
}

// RateLimit 杩斿洖鍩轰簬 IP 鐨勪护鐗屾《闄愭祦涓棿浠躲€?
//
// 浣跨敤鏂瑰紡锛氶粯璁ら厤缃嵆鍙紝涔熷彲鑷畾涔夛細
//
//	r.Use(middleware.RateLimit(middleware.RateLimiterConfig{
//	    Rate:  200,
//	    Burst: 50,
//	}))
//
// 瓒呰繃闄愬埗鐨勮姹傝繑鍥?429 Too Many Requests銆?
// 鍝嶅簲浣撲腑鍖呭惈涓氬姟閿欒鐮?CodeTooManyRequests(1005)銆?
//
// 涓轰粈涔?RateLimit 鏀惧湪涓棿浠堕摼鐨勬渶鏈熬锛?
//   鍓嶉潰鐨勪腑闂翠欢锛堣璇併€佽湝缃愩€両P 榛戝悕鍗曪級宸茬粡杩囨护鎺変簡澶ч噺鐨勬伓鎰忚姹傦紝
//   RateLimit 鍙渶瑕佸鐞嗗墿浣欑殑"姝ｅ父浣嗛鐜囪繃楂?鐨勮姹傘€?
//   杩欐牱鍙互缂撳瓨鏇村皯鐨?IP 鍒伴檺娴佸櫒 map 涓€?
func RateLimit(cfg RateLimiterConfig) gin.HandlerFunc {
	limiter := newIPRateLimiter(cfg)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			response.ErrorWithCode(c.Writer, errors.CodeTooManyRequests, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

